package exec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

func TestCreateLink_CreatesSymlink(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "link")

	if err := CreateLink(tgt, src); err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	got, err := os.Readlink(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("Readlink = %q, want %q", got, src)
	}
}

func TestUpdateLink_ReplacesWrongSymlink(t *testing.T) {
	root := t.TempDir()
	oldSrc := filepath.Join(root, "old")
	newSrc := filepath.Join(root, "new")
	if err := os.Mkdir(oldSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(newSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "link")
	if err := os.Symlink(oldSrc, tgt); err != nil {
		t.Fatal(err)
	}

	if err := UpdateLink(tgt, newSrc); err != nil {
		t.Fatalf("UpdateLink: %v", err)
	}
	got, err := os.Readlink(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if got != newSrc {
		t.Fatalf("Readlink = %q, want %q", got, newSrc)
	}
}

func TestApplyFileLinks_CreateAndUpdate(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	srcA := filepath.Join(configDir, "a")
	srcB := filepath.Join(configDir, "b")
	if err := os.MkdirAll(srcA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcB, 0o755); err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(root, "wrong")
	if err := os.Mkdir(wrong, 0o755); err != nil {
		t.Fatal(err)
	}

	createTgt := filepath.Join(root, "create-link")
	updateTgt := filepath.Join(root, "update-link")
	if err := os.Symlink(wrong, updateTgt); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionLinkCreate, Name: createTgt, Source: "a"},
		{Type: plan.ActionLinkUpdate, Name: updateTgt, Source: "b"},
	}}

	n, err := ApplyFileLinks(p, configDir, nil)
	if err != nil {
		t.Fatalf("ApplyFileLinks: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}

	if got, err := os.Readlink(createTgt); err != nil || got != srcA {
		t.Fatalf("create link = %q err=%v, want %q", got, err, srcA)
	}
	if got, err := os.Readlink(updateTgt); err != nil || got != srcB {
		t.Fatalf("update link = %q err=%v, want %q", got, err, srcB)
	}
}

func TestApplyFileLinks_FailsIfTargetIsRegularFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	src := filepath.Join(configDir, "a")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "file")
	if err := os.WriteFile(tgt, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: tgt, Source: "a"},
	}}
	if _, err := ApplyFileLinks(p, configDir, nil); err == nil {
		t.Fatal("expected error for regular file target")
	}
}

func TestCreateLink_CreatesParentDirs(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "nested", "dir", "link")
	if err := CreateLink(tgt, src); err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	got, err := os.Readlink(tgt)
	if err != nil || got != src {
		t.Fatalf("Readlink = %q err=%v", got, err)
	}
}

func TestApplyFileLinks_ContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	srcOK := filepath.Join(configDir, "ok")
	srcBad := filepath.Join(configDir, "bad")
	if err := os.MkdirAll(srcOK, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcBad, 0o755); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	okTgt := filepath.Join(root, "missing-parent", "zshrc")
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: blocked, Source: "bad"},
		{Type: plan.ActionLinkCreate, Name: okTgt, Source: "ok"},
	}}
	n, err := ApplyFileLinks(p, configDir, nil)
	if err == nil {
		t.Fatal("expected error from blocked target")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1 successful link after failure", n)
	}
	if got, err := os.Readlink(okTgt); err != nil || got != srcOK {
		t.Fatalf("ok link = %q err=%v, want %q", got, err, srcOK)
	}
}
