package plan

import (
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/pam"
)

func TestExpandPAMFormulae_AddsReattachOnly(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		Reattach:    true,
		WatchIDAuth: true,
	}
	got := ExpandPAMFormulae(config.Packages{Formulae: []string{"git"}}, cfg)
	want := []string{"git", "pam-reattach"}
	if !equalStrings(got.Formulae, want) {
		t.Fatalf("Formulae = %v, want %v (pam-watchid is not a brew core formula)", got.Formulae, want)
	}
}

func TestExpandPAMFormulae_SkipsAlreadyDesired(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured: true,
		Enable:     true,
		Reattach:   true,
	}
	got := ExpandPAMFormulae(config.Packages{Formulae: []string{"pam-reattach", "fzf"}}, cfg)
	if !equalStrings(got.Formulae, []string{"pam-reattach", "fzf"}) {
		t.Fatalf("Formulae = %v, want unchanged order without dupes", got.Formulae)
	}
}

func TestExpandPAMFormulae_OmittedOrDisabled(t *testing.T) {
	pkgs := config.Packages{Formulae: []string{"git"}}
	if got := ExpandPAMFormulae(pkgs, config.SudoLocalPAM{}); !equalStrings(got.Formulae, pkgs.Formulae) {
		t.Fatalf("unconfigured: got %v", got.Formulae)
	}
	disabled := config.SudoLocalPAM{Configured: true, Enable: false, Reattach: true}
	if got := ExpandPAMFormulae(pkgs, disabled); !equalStrings(got.Formulae, pkgs.Formulae) {
		t.Fatalf("disabled: got %v", got.Formulae)
	}
}

func TestBuildPAMPlan_EnableWriteAndInclude(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		Reattach:    true,
		TouchIDAuth: true,
		WatchIDAuth: true,
	}
	reattach := "/opt/homebrew/lib/pam/pam_reattach.so"
	watchid := "/opt/homebrew/lib/pam/pam_watchid.so"
	desired := pam.RenderSudoLocal(cfg, reattach, watchid)

	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/test-pam.d/sudo_local",
		SudoPath:         "/tmp/test-pam.d/sudo",
		SudoLocalContent: nil,
		SudoLocalExists:  false,
		SudoContent:      []byte("# sudo\nauth required pam_opendirectory.so\n"),
		ReattachPath:     reattach,
		WatchIDPath:      watchid,
	})

	types := ActionTypes(p)
	if len(types) != 2 || types[0] != ActionPAMSudoLocalWrite || types[1] != ActionPAMSudoInclude {
		t.Fatalf("types = %v, want [pam_sudo_local_write pam_sudo_include]", types)
	}
	write := p.Actions[0]
	if write.Name != "/tmp/test-pam.d/sudo_local" || write.Value != desired {
		t.Fatalf("write action = %+v, want Name=path Value=desired body", write)
	}
	if p.Actions[1].Name != "/tmp/test-pam.d/sudo" {
		t.Fatalf("include Name = %q", p.Actions[1].Name)
	}
}

func TestBuildPAMPlan_EnablePlusImpliedFormulaInstalls(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		Reattach:    true,
		WatchIDAuth: true,
	}
	pkgs := ExpandPAMFormulae(config.Packages{}, cfg)
	brewPlan := BuildBrewPlan(pkgs, discovery.BrewState{})
	pamPlan := BuildPAMPlan(PAMDiffInput{
		Config:        cfg,
		SudoLocalPath: "/etc/pam.d/sudo_local",
		SudoPath:      "/etc/pam.d/sudo",
		SudoContent:   []byte("auth include sudo_local\n"),
		ReattachPath:  "/p/lib/pam/pam_reattach.so",
		WatchIDPath:   "/p/lib/pam/pam_watchid.so",
	})
	merged := MergePlans(brewPlan, pamPlan)

	formulae := ActionNames(merged, ActionFormulaInstall)
	if !containsAll(formulae, "pam-reattach") {
		t.Fatalf("formula installs = %v, want pam-reattach", formulae)
	}
	if containsAll(formulae, "pam-watchid") {
		t.Fatalf("formula installs = %v, must not auto-add pam-watchid", formulae)
	}
	if types := ActionTypes(pamPlan); len(types) != 1 || types[0] != ActionPAMSudoLocalWrite {
		t.Fatalf("pam types = %v, want write only (include present)", types)
	}
}

func TestBuildPAMPlan_EnableFalseManagedWritesStub(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: false}
	managed := pam.ManagedMarker + "\nauth sufficient pam_tid.so\n"
	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/sudo_local",
		SudoLocalContent: []byte(managed),
		SudoLocalExists:  true,
	})
	if types := ActionTypes(p); len(types) != 1 || types[0] != ActionPAMSudoLocalWrite {
		t.Fatalf("types = %v, want [pam_sudo_local_write]", types)
	}
	if p.Actions[0].Name != "/tmp/sudo_local" {
		t.Fatalf("Name = %q", p.Actions[0].Name)
	}
	if p.Actions[0].Value != pam.DisabledSudoLocal {
		t.Fatalf("Value = %q, want disabled stub", p.Actions[0].Value)
	}
}

func TestBuildPAMPlan_EnableFalseMissingWritesStub(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: false}
	p := BuildPAMPlan(PAMDiffInput{
		Config:          cfg,
		SudoLocalPath:   "/tmp/sudo_local",
		SudoLocalExists: false,
	})
	if types := ActionTypes(p); len(types) != 1 || types[0] != ActionPAMSudoLocalWrite {
		t.Fatalf("types = %v, want write stub when missing", types)
	}
	if p.Actions[0].Value != pam.DisabledSudoLocal {
		t.Fatalf("Value = %q, want stub", p.Actions[0].Value)
	}
}

func TestBuildPAMPlan_EnableFalseAlreadyStubNoop(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: false}
	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/sudo_local",
		SudoLocalContent: []byte(pam.DisabledSudoLocal),
		SudoLocalExists:  true,
	})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want none when already stub", p.Actions)
	}
}

func TestBuildPAMPlan_EnableFalseUnmanagedNoWrite(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: false}
	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/sudo_local",
		SudoLocalContent: []byte("auth sufficient pam_tid.so\n"),
		SudoLocalExists:  true,
	})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want none for unmanaged file", p.Actions)
	}
}

func TestBuildPAMPlan_OmittedNoActions(t *testing.T) {
	p := BuildPAMPlan(PAMDiffInput{
		Config:           config.SudoLocalPAM{Enable: true, TouchIDAuth: true},
		SudoLocalPath:    "/tmp/sudo_local",
		SudoPath:         "/tmp/sudo",
		SudoLocalContent: []byte("x"),
		SudoLocalExists:  true,
		SudoContent:      []byte("auth required pam_opendirectory.so\n"),
	})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want none when Configured=false", p.Actions)
	}
}

func TestBuildPAMPlan_NoopWhenAlreadyDesired(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: true, TouchIDAuth: true}
	body := pam.RenderSudoLocal(cfg, "", "")
	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/sudo_local",
		SudoPath:         "/tmp/sudo",
		SudoLocalContent: []byte(body),
		SudoLocalExists:  true,
		SudoContent:      []byte("auth       include        sudo_local\n"),
	})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want none when already reconciled", p.Actions)
	}
}

func TestBuildPAMPlan_SudoMissingInclude(t *testing.T) {
	cfg := config.SudoLocalPAM{Configured: true, Enable: true, TouchIDAuth: true}
	body := pam.RenderSudoLocal(cfg, "", "")
	p := BuildPAMPlan(PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    "/tmp/sudo_local",
		SudoPath:         "/tmp/sudo",
		SudoLocalContent: []byte(body),
		SudoLocalExists:  true,
		SudoContent:      []byte("# sudo\nauth required pam_opendirectory.so\n"),
	})
	if types := ActionTypes(p); len(types) != 1 || types[0] != ActionPAMSudoInclude {
		t.Fatalf("types = %v, want [pam_sudo_include]", types)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsAll(have []string, want ...string) bool {
	set := make(map[string]struct{}, len(have))
	for _, h := range have {
		set[strings.ToLower(h)] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[strings.ToLower(w)]; !ok {
			return false
		}
	}
	return true
}
