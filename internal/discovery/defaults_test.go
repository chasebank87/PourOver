package discovery

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

type fakeDefaults struct {
	values map[string]string // domain\x00key -> raw
	calls  [][]string
}

func (f *fakeDefaults) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{}, args...))
	if len(args) >= 3 && args[0] == "read" {
		key := args[1] + "\x00" + args[2]
		if v, ok := f.values[key]; ok {
			return []byte(v), nil
		}
		return nil, &DefaultsExitError{
			Args:   args,
			Stderr: fmt.Sprintf("The domain/default pair of (%s, %s) does not exist", args[1], args[2]),
			Err:    fmt.Errorf("exit status 1"),
		}
	}
	return nil, fmt.Errorf("unexpected %v", args)
}

func TestParseDefaultsRead(t *testing.T) {
	b, err := ParseDefaultsRead("1", config.SettingBool)
	if err != nil || !b.Bool {
		t.Fatalf("bool = %#v err=%v", b, err)
	}
	i, err := ParseDefaultsRead("48", config.SettingInt)
	if err != nil || i.Int != 48 {
		t.Fatalf("int = %#v err=%v", i, err)
	}
	s, err := ParseDefaultsRead(`"left"`, config.SettingString)
	if err != nil || s.String != "left" {
		t.Fatalf("string = %#v err=%v", s, err)
	}
}

func TestSettingValuesEqual_HomeExpansion(t *testing.T) {
	a := config.SettingValue{Kind: config.SettingString, String: "~/Desktop"}
	b := config.SettingValue{Kind: config.SettingString, String: "~/Desktop"}
	if !SettingValuesEqual(a, b) {
		t.Fatal("expected equal")
	}
}

func TestParseDefaultsRead_Array(t *testing.T) {
	raw := `(
        {
            tile-data = {
                file-data = {
                    "_CFURLString" = "file:///Applications/Safari.app/";
                };
            };
        }
)`
	v, err := ParseDefaultsRead(raw, config.SettingArray)
	if err != nil {
		t.Fatal(err)
	}
	if v.Kind != config.SettingArray || len(v.Array) != 1 || v.Array[0] != "/Applications/Safari.app" {
		t.Fatalf("got %#v", v)
	}
}

func TestSettingValuesEqual_DockArrays(t *testing.T) {
	a := config.SettingValue{Kind: config.SettingArray, Array: []string{"/Applications/Safari.app/"}}
	b := config.SettingValue{Kind: config.SettingArray, Array: []string{"/Applications/Safari.app"}}
	if !SettingValuesEqual(a, b) {
		t.Fatal("expected trailing-slash-normalized equal")
	}
	c := config.SettingValue{Kind: config.SettingArray, Array: []string{"/Applications/Mail.app"}}
	if SettingValuesEqual(a, c) {
		t.Fatal("expected drift")
	}
}

func TestDiscoverDefaults_DriftAndMatch(t *testing.T) {
	fake := &fakeDefaults{values: map[string]string{
		config.DomainDock + "\x00autohide": "1",
		config.DomainDock + "\x00tilesize": "64",
	}}
	desired := []config.DesiredSetting{
		{Domain: config.DomainDock, Key: "autohide", Value: config.SettingValue{Kind: config.SettingBool, Bool: true}, Section: "dock"},
		{Domain: config.DomainDock, Key: "tilesize", Value: config.SettingValue{Kind: config.SettingInt, Int: 48}, Section: "dock"},
		{Domain: config.DomainDock, Key: "orientation", Value: config.SettingValue{Kind: config.SettingString, String: "left"}, Section: "dock"},
	}
	st, err := DiscoverDefaults(context.Background(), fake, desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 3 {
		t.Fatalf("len=%d", len(st))
	}
	if st[0].Drift {
		t.Fatal("autohide should match")
	}
	if !st[1].Drift {
		t.Fatal("tilesize should drift")
	}
	if !st[2].Drift || st[2].Found {
		t.Fatalf("orientation missing: %#v", st[2])
	}
}

func TestDiscoverDefaults_ReadError(t *testing.T) {
	bad := defaultsFunc(func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, &DefaultsExitError{Args: args, Stderr: "permission denied", Err: fmt.Errorf("exit status 1")}
	})
	_, err := DiscoverDefaults(context.Background(), bad, []config.DesiredSetting{
		{Domain: "x", Key: "y", Value: config.SettingValue{Kind: config.SettingBool, Bool: true}},
	})
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("err=%v", err)
	}
}

type defaultsFunc func(ctx context.Context, args ...string) ([]byte, error)

func (f defaultsFunc) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	return f(ctx, args...)
}
