package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"1.0.0", "dev", true},
		{"0.1.1", "0.1.0", true},
	}
	for _, tc := range cases {
		if got := isNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("isNewer(%q,%q)=%v want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckAndApply_UpdatesBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pourover")
	if err := os.WriteFile(exe, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	archive := buildTestArchive(t, []byte("new-binary"))
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/releases/latest"):
			body := `{
  "tag_name": "v0.2.0",
  "assets": [
    {"name": "pourover_0.2.0_darwin_arm64.tar.gz", "browser_download_url": "https://example.com/pourover_0.2.0_darwin_arm64.tar.gz"},
    {"name": "pourover_0.2.0_darwin_amd64.tar.gz", "browser_download_url": "https://example.com/pourover_0.2.0_darwin_amd64.tar.gz"}
  ]
}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		case strings.HasSuffix(req.URL.Path, ".tar.gz"):
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(archive)),
				Header:     make(http.Header),
			}, nil
		default:
			t.Fatalf("unexpected URL %s", req.URL)
			return nil, nil
		}
	})

	var out strings.Builder
	res, err := CheckAndApply(Options{
		Repo:       "chasebank87/PourOver",
		Current:    "0.1.0",
		Executable: exe,
		Client:     client,
		Stdout:     &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.LatestTag != "0.2.0" {
		t.Fatalf("result = %+v", res)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("binary = %q", got)
	}
	alias := filepath.Join(dir, "pour")
	target, err := os.Readlink(alias)
	if err != nil {
		t.Fatalf("pour alias: %v", err)
	}
	if target != "pourover" {
		t.Fatalf("pour -> %q, want pourover", target)
	}
}

func TestEnsurePourAlias_SkipsRegularFile(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pourover")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "pour")
	if err := os.WriteFile(alias, []byte("other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ensurePourAlias(exe); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(alias)
	if err != nil || string(data) != "other" {
		t.Fatalf("clobbered pour file: %q err=%v", data, err)
	}
}

func TestCheckAndApply_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pourover")
	if err := os.WriteFile(exe, []byte("same"), 0o755); err != nil {
		t.Fatal(err)
	}
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v0.1.0","assets":[{"name":"pourover_0.1.0_darwin_arm64.tar.gz","browser_download_url":"https://example.com/a.tar.gz"},{"name":"pourover_0.1.0_darwin_amd64.tar.gz","browser_download_url":"https://example.com/b.tar.gz"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	res, err := CheckAndApply(Options{
		Current:    "0.1.0",
		Executable: exe,
		Client:     client,
		Stdout:     io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated {
		t.Fatal("should not update when already current")
	}
}

func buildTestArchive(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "pourover", Mode: 0o755, Size: int64(len(payload))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
