// Package server expose l'interface graphique de VaultKeeper : une API JSON
// et l'application React embarquée, servies uniquement sur 127.0.0.1.
//
// Sécurité :
//   - écoute sur la boucle locale uniquement ;
//   - jeton aléatoire généré à chaque lancement, exigé sur toute l'API
//     (empêche un site web ouvert dans le navigateur d'appeler l'API) ;
//   - vérification de l'en-tête Host (protection contre le DNS rebinding) ;
//   - aucune valeur secrète ne transite : l'API ne renvoie que l'inventaire.
package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/digitaleflex/vaultkeeper/internal/scanner"
)

type Server struct {
	Version string
	Token   string
	static  fs.FS
	addr    string

	mu     sync.Mutex
	last   *scanner.Report
	busy   bool
	scanFn func(scanner.Options) (*scanner.Report, error)
}

// New crée un serveur ; static est le dossier "dist" de l'application web.
func New(version string, static fs.FS) (*Server, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return &Server{Version: version, Token: hex.EncodeToString(b), static: static, scanFn: scanner.Scan}, nil
}

// Listen ouvre le port (0 = port libre aléatoire) et renvoie l'URL à ouvrir.
func (s *Server) Listen(port int) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
	if err != nil {
		return nil, "", err
	}
	s.addr = ln.Addr().String()
	return ln, "http://" + s.addr + "/#token=" + s.Token, nil
}

func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return srv.Serve(ln)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/info", s.auth(s.handleInfo))
	mux.HandleFunc("GET /api/report", s.auth(s.handleReport))
	mux.HandleFunc("POST /api/scan", s.auth(s.handleScan))
	mux.Handle("/", s.spa())
	return s.guard(mux)
}

// guard : Host autorisé + en-têtes de sécurité.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.addr != "" {
			_, port, _ := net.SplitHostPort(s.addr)
			if r.Host != "127.0.0.1:"+port && r.Host != "localhost:"+port {
				http.Error(w, "hôte non autorisé", http.StatusForbidden)
				return
			}
		}
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-VK-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "jeton invalide : relancez `vk ui`"})
			return
		}
		next(w, r)
	}
}

type info struct {
	Version     string   `json:"version"`
	Host        string   `json:"host"`
	OS          string   `json:"os"`
	Home        string   `json:"home"`
	Suggestions []string `json:"suggestions"`
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	host, _ := os.Hostname()
	var sugg []string
	for _, d := range []string{"code", "Code", "dev", "projects", "Projects", "projets", "Projets", "workspace", "src", "repos", "Documents/GitHub", "go/src"} {
		p := filepath.Join(home, d)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			sugg = append(sugg, p)
		}
	}
	writeJSON(w, http.StatusOK, info{Version: s.Version, Host: host, OS: osName(), Home: home, Suggestions: sugg})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, s.last)
}

type scanRequest struct {
	Paths       []string `json:"paths"`
	Depth       int      `json:"depth"`
	SkipHistory bool     `json:"skipHistory"`
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "requête invalide"})
		return
	}
	var paths []string
	for _, p := range req.Paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "dossier introuvable : " + p})
			return
		}
		paths = append(paths, p)
	}
	if req.Depth < 1 || req.Depth > 20 {
		req.Depth = 6
	}

	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "une analyse est déjà en cours"})
		return
	}
	s.busy = true
	s.mu.Unlock()

	rep, err := s.scanFn(scanner.Options{ProjectDirs: paths, MaxDepth: req.Depth, SkipHistory: req.SkipHistory})

	s.mu.Lock()
	s.busy = false
	if err == nil {
		s.last = rep
	}
	s.mu.Unlock()

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// spa sert les fichiers statiques, avec repli sur index.html.
func (s *Server) spa() http.Handler {
	files := http.FileServer(http.FS(s.static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if _, err := fs.Stat(s.static, p); errors.Is(err, fs.ErrNotExist) {
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
