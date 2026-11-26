package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/digitaleflex/vaultkeeper/internal/scanner"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New("test", fstest.MapFS{"index.html": {Data: []byte("<html>app</html>")}})
	if err != nil {
		t.Fatal(err)
	}
	s.addr = "127.0.0.1:4242"
	s.scanFn = func(o scanner.Options) (*scanner.Report, error) {
		return &scanner.Report{Host: "pc", Findings: []scanner.Finding{{ID: "a", Type: "Clé SSH"}}}, nil
	}
	s.checkFn = func() updateInfo { return updateInfo{Current: "test", Latest: "9.9.9", Available: true} }
	return s
}

func do(s *Server, method, path, host, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = host
	if token != "" {
		req.Header.Set("X-VK-Token", token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestAPIRequiresToken(t *testing.T) {
	s := newTestServer(t)
	if rec := do(s, "GET", "/api/info", "127.0.0.1:4242", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sans jeton : attendu 401, obtenu %d", rec.Code)
	}
	if rec := do(s, "GET", "/api/info", "127.0.0.1:4242", "mauvais", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("mauvais jeton : attendu 401, obtenu %d", rec.Code)
	}
	if rec := do(s, "GET", "/api/info", "127.0.0.1:4242", s.Token, ""); rec.Code != http.StatusOK {
		t.Fatalf("bon jeton : attendu 200, obtenu %d", rec.Code)
	}
}

func TestRejectsForeignHost(t *testing.T) {
	s := newTestServer(t)
	if rec := do(s, "GET", "/", "evil.example.com:4242", "", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("DNS rebinding : attendu 403, obtenu %d", rec.Code)
	}
	if rec := do(s, "GET", "/", "localhost:4242", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("localhost : attendu 200, obtenu %d", rec.Code)
	}
}

func TestScanFlow(t *testing.T) {
	s := newTestServer(t)
	if rec := do(s, "GET", "/api/report", "127.0.0.1:4242", s.Token, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("avant analyse : attendu 204, obtenu %d", rec.Code)
	}
	if rec := do(s, "POST", "/api/scan", "127.0.0.1:4242", s.Token, `{"paths":["/dossier/inexistant"]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("dossier inexistant : attendu 400, obtenu %d", rec.Code)
	}
	rec := do(s, "POST", "/api/scan", "127.0.0.1:4242", s.Token, `{"paths":[],"depth":4}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Clé SSH") {
		t.Fatalf("analyse : %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(s, "GET", "/api/report", "127.0.0.1:4242", s.Token, ""); rec.Code != http.StatusOK {
		t.Fatalf("après analyse : attendu 200, obtenu %d", rec.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	s := newTestServer(t)
	rec := do(s, "GET", "/inventaire", "127.0.0.1:4242", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app") {
		t.Fatalf("repli SPA : %d %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateEndpoint(t *testing.T) {
	s := newTestServer(t)
	rec := do(s, "GET", "/api/update", "127.0.0.1:4242", s.Token, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"available":true`) {
		t.Fatalf("update : %d %s", rec.Code, rec.Body.String())
	}
}
