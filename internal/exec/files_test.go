package exec

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
	tmpl "github.com/chasebank87/PourOver/internal/template"
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
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcA, []byte("content-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("content-b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	createTgt := filepath.Join(root, "create-link")
	updateTgt := filepath.Join(root, "update-link")
	if err := os.Symlink(filepath.Join(root, "wrong"), updateTgt); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionLinkCreate, Name: createTgt, Source: "a"},
		{Type: plan.ActionLinkUpdate, Name: updateTgt, Source: "b"},
	}}

	n, err := ApplyFileLinks(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err != nil {
		t.Fatalf("ApplyFileLinks: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}

	gotA, err := os.ReadFile(createTgt)
	if err != nil || string(gotA) != "content-a\n" {
		t.Fatalf("create file = %q err=%v", gotA, err)
	}
	info, err := os.Lstat(updateTgt)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("update target should be a regular file, not a symlink")
	}
	gotB, err := os.ReadFile(updateTgt)
	if err != nil || string(gotB) != "content-b\n" {
		t.Fatalf("update file = %q err=%v", gotB, err)
	}
}

func TestApplyFileLinks_OverwritesRegularFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "a"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "file")
	if err := os.WriteFile(tgt, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: tgt, Source: "a"},
	}}
	if _, err := ApplyFileLinks(p, FileApplyOptions{ConfigDir: configDir}, nil); err != nil {
		t.Fatalf("ApplyFileLinks: %v", err)
	}
	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "new\n" {
		t.Fatalf("got %q err=%v", got, err)
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
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "ok"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirBlocked := filepath.Join(root, "blocked-dir")
	if err := os.Mkdir(dirBlocked, 0o755); err != nil {
		t.Fatal(err)
	}
	okTgt := filepath.Join(root, "missing-parent", "zshrc")
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: dirBlocked, Source: "ok"},
		{Type: plan.ActionLinkCreate, Name: okTgt, Source: "ok"},
	}}
	n, err := ApplyFileLinks(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err == nil {
		t.Fatal("expected error for directory target")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}
	got, err := os.ReadFile(okTgt)
	if err != nil || string(got) != "ok\n" {
		t.Fatalf("ok target = %q err=%v", got, err)
	}
}

func TestApplyManagedCopies_CreatesAndOverwrites(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	srcDir := filepath.Join(configDir, "config")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcA := filepath.Join(srcDir, "a.conf")
	srcB := filepath.Join(srcDir, "b.conf")
	if err := os.WriteFile(srcA, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}

	createTgt := filepath.Join(root, "home", "nested", "a.conf")
	overwriteTgt := filepath.Join(root, "home", "b.conf")
	if err := os.MkdirAll(filepath.Dir(overwriteTgt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overwriteTgt, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionManagedCopy, Name: createTgt, Source: "config/a.conf"},
		{Type: plan.ActionManagedCopy, Name: overwriteTgt, Source: "config/b.conf"},
	}}

	n, err := ApplyManagedCopies(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err != nil {
		t.Fatalf("ApplyManagedCopies: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}

	if got, err := os.ReadFile(createTgt); err != nil || string(got) != "alpha" {
		t.Fatalf("create = %q err=%v, want alpha", got, err)
	}
	if got, err := os.ReadFile(overwriteTgt); err != nil || string(got) != "bravo" {
		t.Fatalf("overwrite = %q err=%v, want bravo", got, err)
	}
	info, err := os.Lstat(createTgt)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("managed copy must write a regular file, not a symlink")
	}
}

func TestApplyManagedCopies_ReplacesSymlinkWithFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "src.txt")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(root, "elsewhere")
	if err := os.WriteFile(elsewhere, []byte("linked"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "target.txt")
	if err := os.Symlink(elsewhere, tgt); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionManagedCopy, Name: tgt, Source: "src.txt"},
	}}
	n, err := ApplyManagedCopies(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err != nil {
		t.Fatalf("ApplyManagedCopies: %v", err)
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}
	info, err := os.Lstat(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("target should no longer be a symlink")
	}
	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "content" {
		t.Fatalf("got %q err=%v", got, err)
	}
	// Original symlink destination must remain untouched.
	if got, err := os.ReadFile(elsewhere); err != nil || string(got) != "linked" {
		t.Fatalf("elsewhere = %q err=%v", got, err)
	}
}

func TestApplyManagedCopies_ContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	okSrc := filepath.Join(configDir, "ok.txt")
	if err := os.WriteFile(okSrc, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirTgt := filepath.Join(root, "isdir")
	if err := os.Mkdir(dirTgt, 0o755); err != nil {
		t.Fatal(err)
	}
	okTgt := filepath.Join(root, "ok.txt")

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionManagedCopy, Name: dirTgt, Source: "ok.txt"},
		{Type: plan.ActionManagedCopy, Name: okTgt, Source: "ok.txt"},
	}}
	n, err := ApplyManagedCopies(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err == nil {
		t.Fatal("expected error for directory target")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1 successful copy after failure", n)
	}
	if got, err := os.ReadFile(okTgt); err != nil || string(got) != "ok" {
		t.Fatalf("ok = %q err=%v", got, err)
	}
}

func TestApplyFileUnlinks_RemovesSymlinkAndFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkTarget := filepath.Join(root, "link-dest")
	if err := os.WriteFile(linkTarget, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "link.txt")
	if err := os.Symlink(linkTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFileUnlink, Name: filePath},
		{Type: plan.ActionFileUnlink, Name: linkPath},
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
	}}
	n, err := ApplyFileUnlinks(p, nil)
	if err != nil {
		t.Fatalf("ApplyFileUnlinks: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}
	if _, err := os.Lstat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link still exists: %v", err)
	}
	if _, err := os.Stat(linkTarget); err != nil {
		t.Fatalf("link destination should remain: %v", err)
	}
}

func TestApplyFileUnlinks_RefusesDirectory(t *testing.T) {
	root := t.TempDir()
	dirPath := filepath.Join(root, "dir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	okFile := filepath.Join(root, "ok.txt")
	if err := os.WriteFile(okFile, []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFileUnlink, Name: dirPath},
		{Type: plan.ActionFileUnlink, Name: okFile},
	}}
	n, err := ApplyFileUnlinks(p, nil)
	if err == nil {
		t.Fatal("expected error refusing directory unlink")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}
	if _, err := os.Stat(dirPath); err != nil {
		t.Fatalf("directory should remain: %v", err)
	}
	if _, err := os.Lstat(okFile); !os.IsNotExist(err) {
		t.Fatalf("ok file should be removed: %v", err)
	}
}

func TestApplyFileLinks_ReplaceBacksUpThenWrites(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	src := filepath.Join(configDir, "zshrc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export NEW=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "home", ".zshrc")
	if err := os.MkdirAll(filepath.Dir(tgt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tgt, []byte("old-zshrc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 8, 15, 5, 30, 0, 0, time.UTC)
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkReplace, Name: tgt, Source: "zshrc"},
	}}
	n, err := ApplyFileLinks(p, FileApplyOptions{
		ConfigDir: configDir,
		StateDir:  stateDir,
		Now:       func() time.Time { return at },
	}, nil)
	if err != nil {
		t.Fatalf("ApplyFileLinks: %v", err)
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}

	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "export NEW=1\n" {
		t.Fatalf("target = %q err=%v", got, err)
	}
	info, err := os.Lstat(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("target should be a regular file")
	}

	backup := filepath.Join(stateDir, "backups", "files", "20260815T053000Z", EscapeBackupPath(tgt))
	data, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("backup missing at %s: %v", backup, err)
	}
	if string(data) != "old-zshrc\n" {
		t.Fatalf("backup content = %q", data)
	}
}

func TestApplyFileLinks_MaterializesDirectorySymlinkRoot(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	srcDir := filepath.Join(configDir, "nvim")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "init.lua"), []byte("-- init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	liveRoot := filepath.Join(root, "home", ".config")
	if err := os.MkdirAll(liveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	liveNvim := filepath.Join(liveRoot, "nvim")
	if err := os.Symlink(srcDir, liveNvim); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(liveNvim, "init.lua")
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkUpdate, Name: target, Source: "nvim/init.lua"},
	}}
	n, err := ApplyFileLinks(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	info, err := os.Lstat(liveNvim)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("nvim root should be a real directory after apply")
	}
	if !info.IsDir() {
		t.Fatal("nvim root should be a directory")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "-- init\n" {
		t.Fatalf("got %q err=%v", got, err)
	}
	src, err := os.ReadFile(filepath.Join(srcDir, "init.lua"))
	if err != nil || string(src) != "-- init\n" {
		t.Fatalf("source mutated: %q err=%v", src, err)
	}
}

func TestApplyManagedCopies_BackupUnexpectedDirectory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "foo.conf")
	if err := os.WriteFile(src, []byte("managed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "home", "foo.conf")
	if err := os.MkdirAll(tgt, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(tgt, "was-dir.txt")
	if err := os.WriteFile(marker, []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionManagedCopy, Name: tgt, Source: "foo.conf", Kind: "backup"},
	}}
	n, err := ApplyManagedCopies(p, FileApplyOptions{
		ConfigDir:   configDir,
		StateDir:    stateDir,
		FileReplace: config.FileReplaceBackup,
		Now:         func() time.Time { return at },
	}, nil)
	if err != nil {
		t.Fatalf("ApplyManagedCopies: %v", err)
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}

	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "managed\n" {
		t.Fatalf("target = %q err=%v", got, err)
	}
	backup := filepath.Join(stateDir, "backups", "files", "20260815T060000Z", EscapeBackupPath(tgt))
	info, err := os.Stat(backup)
	if err != nil || !info.IsDir() {
		t.Fatalf("backup dir = %#v err=%v", info, err)
	}
	if data, err := os.ReadFile(filepath.Join(backup, "was-dir.txt")); err != nil || string(data) != "inside\n" {
		t.Fatalf("backed-up contents = %q err=%v", data, err)
	}
}

func TestEscapeBackupPath(t *testing.T) {
	got := EscapeBackupPath("/Users/chase/.config/nvim")
	want := "_Users_chase_.config_nvim"
	if got != want {
		t.Fatalf("EscapeBackupPath = %q, want %q", got, want)
	}
}

func TestApplyFilePrunes_ByMode(t *testing.T) {
	root := t.TempDir()
	pathA := filepath.Join(root, "a.txt")
	pathB := filepath.Join(root, "b.txt")
	write := func(t *testing.T) {
		t.Helper()
		if err := os.WriteFile(pathA, []byte("a"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pathB, []byte("b"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFilePrune, Name: pathA},
		{Type: plan.ActionFilePrune, Name: pathB},
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
	}}

	t.Run("safe_confirm_false", func(t *testing.T) {
		write(t)
		var prompted bool
		confirm := func(paths []string) bool {
			prompted = true
			if len(paths) != 2 {
				t.Fatalf("paths = %v, want 2", paths)
			}
			return false
		}
		removed, err := ApplyFilePrunes(p, config.FilesModeSafe, confirm, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !prompted {
			t.Fatal("expected confirm prompt")
		}
		if len(removed) != 0 {
			t.Fatalf("removed=%v, want none", removed)
		}
		if _, err := os.Stat(pathA); err != nil {
			t.Fatalf("pathA should remain: %v", err)
		}
		if _, err := os.Stat(pathB); err != nil {
			t.Fatalf("pathB should remain: %v", err)
		}
	})

	t.Run("safe_confirm_true", func(t *testing.T) {
		write(t)
		var gotPaths []string
		confirm := func(paths []string) bool {
			gotPaths = append([]string(nil), paths...)
			return true
		}
		removed, err := ApplyFilePrunes(p, config.FilesModeSafe, confirm, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(removed) != 2 || removed[0] != pathA || removed[1] != pathB {
			t.Fatalf("removed=%v, want [%s %s]", removed, pathA, pathB)
		}
		if len(gotPaths) != 2 || gotPaths[0] != pathA || gotPaths[1] != pathB {
			t.Fatalf("confirm paths = %v", gotPaths)
		}
		if _, err := os.Lstat(pathA); !os.IsNotExist(err) {
			t.Fatalf("pathA should be removed: %v", err)
		}
		if _, err := os.Lstat(pathB); !os.IsNotExist(err) {
			t.Fatalf("pathB should be removed: %v", err)
		}
	})

	t.Run("strict_no_confirm", func(t *testing.T) {
		write(t)
		var prompted bool
		confirm := func(paths []string) bool {
			prompted = true
			return false
		}
		removed, err := ApplyFilePrunes(p, config.FilesModeStrict, confirm, nil)
		if err != nil {
			t.Fatal(err)
		}
		if prompted {
			t.Fatal("strict should not prompt")
		}
		if len(removed) != 2 {
			t.Fatalf("removed=%v, want 2", removed)
		}
		if _, err := os.Lstat(pathA); !os.IsNotExist(err) {
			t.Fatalf("pathA should be removed: %v", err)
		}
	})

	t.Run("non_destructive_skips", func(t *testing.T) {
		write(t)
		var prompted bool
		confirm := func(paths []string) bool {
			prompted = true
			return true
		}
		removed, err := ApplyFilePrunes(p, config.FilesModeNonDestructive, confirm, nil)
		if err != nil {
			t.Fatal(err)
		}
		if prompted {
			t.Fatal("non_destructive should not prompt")
		}
		if len(removed) != 0 {
			t.Fatalf("removed=%v, want none", removed)
		}
		if _, err := os.Stat(pathA); err != nil {
			t.Fatalf("pathA should remain: %v", err)
		}
	})

	t.Run("no_prunes_no_prompt", func(t *testing.T) {
		onlyInstall := plan.Plan{Actions: []plan.Action{{Type: plan.ActionFormulaInstall, Name: "fzf"}}}
		var prompted bool
		confirm := func(paths []string) bool {
			prompted = true
			return true
		}
		removed, err := ApplyFilePrunes(onlyInstall, config.FilesModeSafe, confirm, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(removed) != 0 || prompted {
			t.Fatalf("removed=%v prompted=%v, want no work", removed, prompted)
		}
	})
}


func TestApplyTemplateWrites_CreatesAndUpdates(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "host.tmpl")
	if err := os.WriteFile(src, []byte("host={{.Hostname}}\nuser={{.User}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, err := tmpl.DefaultContext()
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("host=%s\nuser=%s\n", ctx.Hostname, ctx.User)

	createTgt := filepath.Join(root, "home", "nested", "out.conf")
	updateTgt := filepath.Join(root, "home", "update.conf")
	if err := os.MkdirAll(filepath.Dir(updateTgt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(updateTgt, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionTemplateWrite, Name: createTgt, Source: "host.tmpl", Value: "--- ignore me\n"},
		{Type: plan.ActionTemplateWrite, Name: updateTgt, Source: "host.tmpl", Value: "diff junk"},
	}}
	n, err := ApplyTemplateWrites(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err != nil {
		t.Fatalf("ApplyTemplateWrites: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}
	for _, tgt := range []string{createTgt, updateTgt} {
		got, err := os.ReadFile(tgt)
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q err=%v, want %q", tgt, got, err, want)
		}
	}
}

func TestApplyTemplateWrites_IgnoresActionValueDiff(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "plain.tmpl")
	if err := os.WriteFile(src, []byte("rendered-live\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "out.txt")
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTemplateWrite, Name: tgt, Source: "plain.tmpl", Value: "this-is-a-diff-not-content"},
	}}
	if _, err := ApplyTemplateWrites(p, FileApplyOptions{ConfigDir: configDir}, nil); err != nil {
		t.Fatalf("ApplyTemplateWrites: %v", err)
	}
	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "rendered-live\n" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestApplyTemplateWrites_ContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "ok.tmpl")
	if err := os.WriteFile(src, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirTgt := filepath.Join(root, "isdir")
	if err := os.Mkdir(dirTgt, 0o755); err != nil {
		t.Fatal(err)
	}
	okTgt := filepath.Join(root, "ok.txt")

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTemplateWrite, Name: dirTgt, Source: "ok.tmpl"},
		{Type: plan.ActionTemplateWrite, Name: okTgt, Source: "ok.tmpl"},
	}}
	n, err := ApplyTemplateWrites(p, FileApplyOptions{ConfigDir: configDir}, nil)
	if err == nil {
		t.Fatal("expected error for directory target")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1 successful write after failure", n)
	}
	if got, err := os.ReadFile(okTgt); err != nil || string(got) != "ok\n" {
		t.Fatalf("ok = %q err=%v", got, err)
	}
}

func TestApplyTemplateWrites_BackupUnexpectedDirectory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(configDir, "foo.tmpl")
	if err := os.WriteFile(src, []byte("from-template\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tgt := filepath.Join(root, "home", "foo.conf")
	if err := os.MkdirAll(tgt, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(tgt, "was-dir.txt")
	if err := os.WriteFile(marker, []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 8, 15, 7, 0, 0, 0, time.UTC)
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTemplateWrite, Name: tgt, Source: "foo.tmpl", Kind: "backup"},
	}}
	n, err := ApplyTemplateWrites(p, FileApplyOptions{
		ConfigDir:   configDir,
		StateDir:    stateDir,
		FileReplace: config.FileReplaceBackup,
		Now:         func() time.Time { return at },
	}, nil)
	if err != nil {
		t.Fatalf("ApplyTemplateWrites: %v", err)
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}
	got, err := os.ReadFile(tgt)
	if err != nil || string(got) != "from-template\n" {
		t.Fatalf("target = %q err=%v", got, err)
	}
	backup := filepath.Join(stateDir, "backups", "files", "20260815T070000Z", EscapeBackupPath(tgt))
	info, err := os.Stat(backup)
	if err != nil || !info.IsDir() {
		t.Fatalf("backup dir = %#v err=%v", info, err)
	}
	if data, err := os.ReadFile(filepath.Join(backup, "was-dir.txt")); err != nil || string(data) != "inside\n" {
		t.Fatalf("backed-up contents = %q err=%v", data, err)
	}
}
