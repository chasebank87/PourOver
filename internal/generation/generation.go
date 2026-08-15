// Package generation builds and loads PourOver activation generations:
// evaluated package/macos snapshots plus content-addressed file blobs.
package generation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	tmpl "github.com/chasebank87/PourOver/internal/template"
)

const (
	generationsDirName = "generations"
	currentFileName    = "current"
	manifestFileName   = "manifest.json"
	filesDirName       = "files"
	// DefaultKeep is how many generations to retain after prune.
	DefaultKeep = 5
)

// FileKind is the declaration kind that produced a generation file entry.
type FileKind string

const (
	FileKindLink     FileKind = "link"
	FileKindManaged  FileKind = "managed"
	FileKindTemplate FileKind = "template"
)

// FileEntry maps a live target path to a content-addressed blob.
type FileEntry struct {
	Target string   `json:"target"`
	Mode   string   `json:"mode"` // e.g. "0644"
	Hash   string   `json:"hash"`
	Kind   FileKind `json:"kind"`
	Source string   `json:"source"` // declaration source (display / debug)
}

// Manifest is the frozen activation artifact (packages + file map).
type Manifest struct {
	ID        string          `json:"id"`
	CreatedAt string          `json:"created_at"`
	Packages  config.Packages `json:"packages"`
	Policy    config.Policy   `json:"policy"`
	MacOS     config.MacOS    `json:"macos"`
	Files     []FileEntry     `json:"files"`
	Unlink    []string        `json:"unlink,omitempty"`
}

// BuildResult is a written generation on disk.
type BuildResult struct {
	Manifest Manifest
	Dir      string
}

// GenerationsDir returns stateDir/generations.
func GenerationsDir(stateDir string) string {
	return filepath.Join(stateDir, generationsDirName)
}

// Dir returns stateDir/generations/<id>.
func Dir(stateDir, id string) string {
	return filepath.Join(GenerationsDir(stateDir), id)
}

// CurrentPath returns stateDir/current.
func CurrentPath(stateDir string) string {
	return filepath.Join(stateDir, currentFileName)
}

// BlobPath returns the path to a content-addressed blob in a generation.
func BlobPath(stateDir, id, hash string) string {
	return filepath.Join(Dir(stateDir, id), filesDirName, hash)
}

// HashBytes returns the SHA-256 hex digest of data.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HashFile hashes the contents of path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Build evaluates file payloads from the manifest into a new generation on disk.
func Build(stateDir, configDir string, m config.Manifest, at time.Time) (BuildResult, error) {
	if strings.TrimSpace(stateDir) == "" {
		return BuildResult{}, fmt.Errorf("generation build: state directory required")
	}
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return BuildResult{}, fmt.Errorf("generation build: config directory: %w", err)
	}

	entries, blobs, err := collectFiles(m, configDir)
	if err != nil {
		return BuildResult{}, err
	}

	id := at.UTC().Format("20060102T150405Z") + "-" + shortHash(entries, m)
	man := Manifest{
		ID:        id,
		CreatedAt: at.UTC().Format(time.RFC3339),
		Packages:  m.Packages,
		Policy:    m.Policy,
		MacOS:     m.MacOS,
		Files:     entries,
		Unlink:    append([]string{}, m.Files.Unlink...),
	}

	genDir := Dir(stateDir, id)
	if err := os.MkdirAll(filepath.Join(genDir, filesDirName), 0o755); err != nil {
		return BuildResult{}, fmt.Errorf("generation mkdir: %w", err)
	}
	for hash, data := range blobs {
		path := filepath.Join(genDir, filesDirName, hash)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return BuildResult{}, fmt.Errorf("generation write blob %s: %w", hash, err)
		}
	}
	if err := writeJSONAtomic(filepath.Join(genDir, manifestFileName), man); err != nil {
		return BuildResult{}, fmt.Errorf("generation write manifest: %w", err)
	}
	return BuildResult{Manifest: man, Dir: genDir}, nil
}

func shortHash(entries []FileEntry, m config.Manifest) string {
	h := sha256.New()
	_ = json.NewEncoder(h).Encode(struct {
		Packages config.Packages
		Policy   config.Policy
		MacOS    config.MacOS
		Files    []FileEntry
		Unlink   []string
	}{m.Packages, m.Policy, m.MacOS, entries, m.Files.Unlink})
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func collectFiles(m config.Manifest, configDir string) ([]FileEntry, map[string][]byte, error) {
	blobs := map[string][]byte{}
	var entries []FileEntry

	for i, link := range m.Files.Links {
		got, err := collectLink(link, configDir, blobs)
		if err != nil {
			return nil, nil, fmt.Errorf("files.links[%d]: %w", i+1, err)
		}
		entries = append(entries, got...)
	}
	for i, mf := range m.Files.Managed {
		e, err := collectManaged(mf, configDir, blobs)
		if err != nil {
			return nil, nil, fmt.Errorf("files.managed[%d]: %w", i+1, err)
		}
		entries = append(entries, e)
	}
	if len(m.Files.Templates) > 0 {
		ctx, err := tmpl.DefaultContext()
		if err != nil {
			return nil, nil, fmt.Errorf("template context: %w", err)
		}
		for i, tf := range m.Files.Templates {
			e, err := collectTemplate(tf, configDir, ctx, blobs)
			if err != nil {
				return nil, nil, fmt.Errorf("files.templates[%d]: %w", i+1, err)
			}
			entries = append(entries, e)
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Target < entries[j].Target })
	return entries, blobs, nil
}

func collectLink(link config.FileLink, configDir string, blobs map[string][]byte) ([]FileEntry, error) {
	sourcePath, err := resolveSource(link.Source, configDir)
	if err != nil {
		return nil, err
	}
	targetRoot, err := resolveTarget(link.Target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source %q does not exist", link.Source)
		}
		return nil, err
	}
	if !info.IsDir() {
		e, err := addBlobFile(sourcePath, targetRoot, FileKindLink, link.Source, blobs)
		if err != nil {
			return nil, err
		}
		return []FileEntry{e}, nil
	}

	var out []FileEntry
	err = filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, rel)
		e, err := addBlobFile(path, target, FileKindLink, filepath.Join(link.Source, rel), blobs)
		if err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func collectManaged(mf config.ManagedFile, configDir string, blobs map[string][]byte) (FileEntry, error) {
	sourcePath, err := resolveSource(mf.Source, configDir)
	if err != nil {
		return FileEntry{}, err
	}
	target, err := resolveTarget(mf.Target)
	if err != nil {
		return FileEntry{}, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileEntry{}, fmt.Errorf("source %q does not exist", mf.Source)
		}
		return FileEntry{}, err
	}
	if info.IsDir() {
		return FileEntry{}, fmt.Errorf("source %q is a directory (managed files must be regular files)", mf.Source)
	}
	return addBlobFile(sourcePath, target, FileKindManaged, mf.Source, blobs)
}

func collectTemplate(tf config.TemplateFile, configDir string, ctx tmpl.Context, blobs map[string][]byte) (FileEntry, error) {
	sourcePath, err := resolveSource(tf.Source, configDir)
	if err != nil {
		return FileEntry{}, err
	}
	target, err := resolveTarget(tf.Target)
	if err != nil {
		return FileEntry{}, err
	}
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileEntry{}, fmt.Errorf("source %q does not exist", tf.Source)
		}
		return FileEntry{}, err
	}
	rendered, err := tmpl.Render(string(src), ctx)
	if err != nil {
		return FileEntry{}, err
	}
	data := []byte(rendered)
	hash := HashBytes(data)
	blobs[hash] = data
	return FileEntry{
		Target: target,
		Mode:   "0644",
		Hash:   hash,
		Kind:   FileKindTemplate,
		Source: tf.Source,
	}, nil
}

func addBlobFile(sourcePath, target string, kind FileKind, sourceDecl string, blobs map[string][]byte) (FileEntry, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return FileEntry{}, err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(sourcePath); err == nil {
		mode = info.Mode().Perm()
	}
	hash := HashBytes(data)
	blobs[hash] = data
	return FileEntry{
		Target: target,
		Mode:   fmt.Sprintf("%04o", mode),
		Hash:   hash,
		Kind:   kind,
		Source: sourceDecl,
	}, nil
}

func resolveSource(source, configDir string) (string, error) {
	expanded, err := expandHome(source)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(configDir, expanded)), nil
}

func resolveTarget(target string) (string, error) {
	expanded, err := expandHome(target)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("target %q must be absolute or start with ~", target)
	}
	return filepath.Clean(expanded), nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// Load reads generations/<id>/manifest.json.
func Load(stateDir, id string) (Manifest, error) {
	path := filepath.Join(Dir(stateDir, id), manifestFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("load generation %s: %w", id, err)
	}
	var man Manifest
	if err := json.Unmarshal(data, &man); err != nil {
		return Manifest{}, fmt.Errorf("parse generation %s: %w", id, err)
	}
	return man, nil
}

// ReadBlob returns blob bytes for hash in generation id.
func ReadBlob(stateDir, id, hash string) ([]byte, error) {
	data, err := os.ReadFile(BlobPath(stateDir, id, hash))
	if err != nil {
		return nil, fmt.Errorf("read generation blob %s/%s: %w", id, hash, err)
	}
	return data, nil
}

// SetCurrent writes stateDir/current with the generation id.
func SetCurrent(stateDir, id string) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	path := CurrentPath(stateDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(id+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Current returns the active generation id, or "" if unset.
func Current(stateDir string) (string, error) {
	data, err := os.ReadFile(CurrentPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Prune deletes old generations, keeping the newest keepCount (and always current).
func Prune(stateDir string, keepCount int) error {
	if keepCount < 1 {
		keepCount = DefaultKeep
	}
	root := GenerationsDir(stateDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	if len(ids) <= keepCount {
		return nil
	}
	current, _ := Current(stateDir)
	drop := ids[:len(ids)-keepCount]
	for _, id := range drop {
		if id == current {
			continue
		}
		_ = os.RemoveAll(Dir(stateDir, id))
	}
	return nil
}

// DeclaredTargets returns absolute targets from the generation file map + unlink.
func DeclaredTargets(man Manifest) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(t string) {
		if t == "" {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	for _, f := range man.Files {
		add(f.Target)
	}
	for _, u := range man.Unlink {
		if abs, err := resolveTarget(u); err == nil {
			add(abs)
		} else {
			add(u)
		}
	}
	sort.Strings(out)
	return out
}

func writeJSONAtomic(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp to %s: %w", path, err)
	}
	return nil
}
