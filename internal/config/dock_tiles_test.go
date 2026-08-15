package config

import (
	"os"
	"strings"
	"testing"
)

func TestEncodeDockPersistentPlist_Apps(t *testing.T) {
	osUserHomeDir = func() (string, error) { return "/Users/test", nil }
	t.Cleanup(func() { osUserHomeDir = os.UserHomeDir })

	plist, err := EncodeDockPersistentPlist(DockPersistentAppsKey, []string{
		"/Applications/Safari.app",
		"~/Applications/Foo.app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plist, "<string>/Applications/Safari.app</string>") {
		t.Fatalf("missing safari path:\n%s", plist)
	}
	if !strings.Contains(plist, "<string>/Users/test/Applications/Foo.app</string>") {
		t.Fatalf("missing expanded home path:\n%s", plist)
	}
	if strings.Contains(plist, "tile-type") {
		t.Fatal("app tiles should not set tile-type")
	}
	if !strings.Contains(plist, "<integer>0</integer>") {
		t.Fatal("app tiles should use _CFURLStringType 0")
	}
}

func TestEncodeDockPersistentPlist_Others(t *testing.T) {
	plist, err := EncodeDockPersistentPlist(DockPersistentOthersKey, []string{
		"/Users/test/Downloads",
		"/Users/test/notes.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plist, "directory-tile") {
		t.Fatalf("expected directory-tile:\n%s", plist)
	}
	if !strings.Contains(plist, "file-tile") {
		t.Fatalf("expected file-tile:\n%s", plist)
	}
	if !strings.Contains(plist, "<string>file:///Users/test/Downloads</string>") {
		t.Fatalf("folder url:\n%s", plist)
	}
	if !strings.Contains(plist, "<integer>15</integer>") {
		t.Fatal("others tiles should use _CFURLStringType 15")
	}
}

func TestExtractDockPersistentPaths(t *testing.T) {
	raw := `(
        {
        tile-data = {
            file-data = {
                "_CFURLString" = "file:///Applications/Safari.app/";
                "_CFURLStringType" = 15;
            };
        };
    },
        {
        tile-data = {
            file-data = {
                _CFURLString = "/System/Applications/Utilities/Terminal.app";
                _CFURLStringType = 0;
            };
        };
    }
)`
	got := ExtractDockPersistentPaths(raw)
	want := []string{"/Applications/Safari.app", "/System/Applications/Utilities/Terminal.app"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
