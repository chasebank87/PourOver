package engine

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/selfupdate"
	"github.com/chasebank87/PourOver/internal/version"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestSelfUpdate_DefaultsCurrentVersion(t *testing.T) {
	prev := version.Version
	version.Version = "0.1.0"
	t.Cleanup(func() { version.Version = prev })

	dir := t.TempDir()
	exe := filepath.Join(dir, "pourover")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	assetName := fmt.Sprintf("pourover_0.2.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive := buildEngineTestArchive(t, []byte("new"))
	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/releases/latest"):
			body := fmt.Sprintf(`{
  "tag_name": "v0.2.0",
  "assets": [
    {"name": %q, "browser_download_url": "https://example.com/%s"}
  ]
}`, assetName, assetName)
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

	res, err := SelfUpdate(selfupdate.Options{
		Executable: exe,
		Client:     client,
		Stdout:     io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.CurrentTag != "0.1.0" || res.LatestTag != "0.2.0" {
		t.Fatalf("result = %+v", res)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("binary = %q", got)
	}
}

func buildEngineTestArchive(t *testing.T, binary []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "pourover", Mode: 0o755, Size: int64(len(binary))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binary); err != nil {
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
