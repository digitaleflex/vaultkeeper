// Package updater met à jour vk depuis les Releases GitHub :
// recherche de la dernière version, téléchargement de l'archive de la
// plateforme, vérification SHA-256, puis remplacement de l'exécutable.
package updater

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const DefaultRepo = "digitaleflex/vaultkeeper"

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"url"` // URL API (fonctionne aussi pour un dépôt privé)
	Size int64  `json:"size"`
}

type Release struct {
	Tag     string  `json:"tag_name"`
	Name    string  `json:"name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

type Client struct {
	APIBase string // https://api.github.com
	Repo    string // owner/name
	Token   string
	HTTP    *http.Client
	GOOS    string
	GOARCH  string
}

// New crée un client ; le jeton vient de GH_TOKEN, GITHUB_TOKEN ou `gh auth token`.
func New() *Client {
	return &Client{
		APIBase: "https://api.github.com",
		Repo:    DefaultRepo,
		Token:   findToken(),
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
	}
}

func findToken() string {
	for _, k := range []string{"VK_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	if path, err := exec.LookPath("gh"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, path, "auth", "token").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

var ErrNoRelease = errors.New("aucune release publiée")

func (c *Client) get(ctx context.Context, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "vaultkeeper-updater")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		switch {
		case res.StatusCode == http.StatusNotFound && c.Token == "":
			return nil, fmt.Errorf("dépôt privé ou introuvable : définissez GH_TOKEN (ou connectez-vous avec `gh auth login`)")
		case res.StatusCode == http.StatusNotFound:
			return nil, ErrNoRelease
		case res.StatusCode == http.StatusUnauthorized:
			return nil, fmt.Errorf("jeton GitHub refusé (expiré ?)")
		}
		return nil, fmt.Errorf("GitHub a répondu %s", res.Status)
	}
	return res, nil
}

// Latest renvoie la dernière release publiée.
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	res, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", c.APIBase, c.Repo), "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r Release
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// AssetName renvoie le nom de l'archive pour cette plateforme.
func (c *Client) AssetName(version string) string {
	label := c.GOOS
	if label == "darwin" {
		label = "macos"
	}
	return fmt.Sprintf("vk_%s_%s_%s.zip", strings.TrimPrefix(version, "v"), label, c.GOARCH)
}

func (c *Client) download(ctx context.Context, a Asset) ([]byte, error) {
	res, err := c.get(ctx, a.URL, "application/octet-stream")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(io.LimitReader(res.Body, 200<<20))
}

// Download récupère l'archive de la plateforme, vérifie sa somme SHA-256
// et renvoie le binaire extrait.
func (c *Client) Download(ctx context.Context, r *Release) ([]byte, error) {
	want := c.AssetName(r.Tag)
	var archive, sums *Asset
	for i := range r.Assets {
		switch r.Assets[i].Name {
		case want:
			archive = &r.Assets[i]
		case "SHA256SUMS":
			sums = &r.Assets[i]
		}
	}
	if archive == nil {
		return nil, fmt.Errorf("pas de paquet %s dans la release %s", want, r.Tag)
	}
	if sums == nil {
		return nil, fmt.Errorf("SHA256SUMS absent de la release %s : mise à jour refusée", r.Tag)
	}
	sumData, err := c.download(ctx, *sums)
	if err != nil {
		return nil, err
	}
	expected := ""
	for _, line := range strings.Split(string(sumData), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == want {
			expected = f[0]
		}
	}
	if expected == "" {
		return nil, fmt.Errorf("%s absent de SHA256SUMS", want)
	}
	data, err := c.download(ctx, *archive)
	if err != nil {
		return nil, err
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != expected {
		return nil, fmt.Errorf("somme SHA-256 invalide pour %s : téléchargement corrompu ou altéré", want)
	}
	return extract(data, c.GOOS)
}

func extract(zipData []byte, goos string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	name := "vk"
	if goos == "windows" {
		name = "vk.exe"
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == name && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, 200<<20))
		}
	}
	return nil, fmt.Errorf("%s introuvable dans l'archive", name)
}

// Replace remplace l'exécutable exe par newBin.
// Sous Windows, un exécutable en cours d'exécution peut être renommé mais
// pas écrasé : l'ancien est donc renommé en .old (supprimé au prochain lancement).
func Replace(exe string, newBin []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".vk-update-*")
	if err != nil {
		return fmt.Errorf("écriture impossible dans %s (droits insuffisants ? essayez avec sudo / en administrateur) : %w", dir, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(newBin); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()
	if err := os.Chmod(tmpName, 0o755); err != nil {
		os.Remove(tmpName)
		return err
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, exe); err != nil {
		_ = os.Rename(old, exe) // retour arrière
		os.Remove(tmpName)
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Remove(old)
	}
	return nil
}

// CleanupOld supprime l'éventuel vk.exe.old laissé par une mise à jour Windows.
func CleanupOld() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}

// Newer indique si latest est strictement plus récente que current (semver simple).
// Une version de développement ("dev", hash git…) est toujours considérée plus ancienne.
func Newer(latest, current string) bool {
	l, okL := parse(latest)
	c, okC := parse(current)
	if !okL {
		return false
	}
	if !okC {
		return true
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
