package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultRepo = "chasebank87/PourOver"

// Release is a subset of the GitHub releases API payload.
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset is a downloadable release file.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// HTTPDoer abstracts http.Client for tests.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Options controls a self-update check/apply.
type Options struct {
	Repo       string // owner/name
	Current    string // current version (e.g. "0.1.0" or "dev")
	Executable string // path to replace; empty = os.Executable()
	Client     HTTPDoer
	Stdout     io.Writer
}

// Result describes what a self-update attempt did.
type Result struct {
	Updated     bool
	CurrentTag  string
	LatestTag   string
	DownloadURL string
}

// CheckAndApply fetches the latest GitHub release and replaces the binary when newer.
func CheckAndApply(opts Options) (Result, error) {
	if opts.Repo == "" {
		opts.Repo = defaultRepo
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 60 * time.Second}
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	exe := opts.Executable
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return Result{}, fmt.Errorf("resolve executable: %w", err)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return Result{}, fmt.Errorf("resolve executable: %w", err)
		}
	}

	release, err := fetchLatest(opts.Client, opts.Repo)
	if err != nil {
		return Result{}, err
	}
	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(opts.Current, "v")
	res := Result{CurrentTag: current, LatestTag: latest}

	if current != "dev" && !isNewer(latest, current) {
		fmt.Fprintf(opts.Stdout, "PourOver is up to date (%s).\n", current)
		return res, nil
	}

	url, err := assetURL(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return res, err
	}
	res.DownloadURL = url
	fmt.Fprintf(opts.Stdout, "Updating PourOver %s -> %s...\n", current, latest)

	if err := replaceFromArchive(opts.Client, url, exe); err != nil {
		return res, err
	}
	res.Updated = true
	fmt.Fprintf(opts.Stdout, "Updated PourOver to %s.\n", latest)
	return res, nil
}

func fetchLatest(client HTTPDoer, repo string) (Release, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pourover-selfupdate")
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Release{}, fmt.Errorf("fetch release: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Release{}, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return Release{}, fmt.Errorf("release missing tag_name")
	}
	return rel, nil
}

func assetURL(rel Release, goos, goarch string) (string, error) {
	// Matches goreleaser name_template: pourover_{version}_darwin_{arch}.tar.gz
	needle := fmt.Sprintf("_%s_%s.tar.gz", goos, goarch)
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, needle) && strings.HasPrefix(a.Name, "pourover_") {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no %s/%s archive in release %s", goos, goarch, rel.TagName)
}

func replaceFromArchive(client HTTPDoer, url, exePath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pourover-selfupdate")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "pourover-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	binPath, err := extractBinary(resp.Body, tmpDir, "pourover")
	if err != nil {
		return err
	}

	info, err := os.Stat(exePath)
	if err != nil {
		return err
	}
	tmpExe := exePath + ".new"
	data, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpExe, data, info.Mode()); err != nil {
		return fmt.Errorf("write temp binary: %w", err)
	}
	if err := os.Rename(tmpExe, exePath); err != nil {
		_ = os.Remove(tmpExe)
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

func extractBinary(r io.Reader, destDir, name string) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		base := filepath.Base(hdr.Name)
		if base != name || hdr.Typeflag != tar.TypeReg {
			continue
		}
		out := filepath.Join(destDir, name)
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("archive missing %s binary", name)
}

// isNewer reports whether latest is strictly newer than current.
// Supports simple dotted numeric versions (1.2.3). Non-numeric segments compare as strings.
func isNewer(latest, current string) bool {
	if current == "" || current == "dev" {
		return true
	}
	if latest == current {
		return false
	}
	lp := strings.Split(latest, ".")
	cp := strings.Split(current, ".")
	n := len(lp)
	if len(cp) > n {
		n = len(cp)
	}
	for i := 0; i < n; i++ {
		var l, c string
		if i < len(lp) {
			l = lp[i]
		}
		if i < len(cp) {
			c = cp[i]
		}
		ln, lErr := atoi(l)
		cn, cErr := atoi(c)
		if lErr == nil && cErr == nil {
			if ln != cn {
				return ln > cn
			}
			continue
		}
		if l != c {
			return l > c
		}
	}
	return false
}

func atoi(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not int")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
