package plan

import (
	"bytes"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/pam"
)

const pamFormulaReattach = "pam-reattach"

// ExpandPAMFormulae appends pam-reattach to desired formulae when reattach is
// enabled and the formula is not already listed. pam-watchid is NOT a Homebrew
// core formula — callers resolve pam_watchid.so via filesystem search instead.
// Call before BuildBrewPlan so implied packages participate in the brew reconcile.
func ExpandPAMFormulae(pkgs config.Packages, cfg config.SudoLocalPAM) config.Packages {
	if !cfg.Configured || !cfg.Enable {
		return pkgs
	}
	out := pkgs
	out.Formulae = append([]string(nil), pkgs.Formulae...)
	have := sliceSet(out.Formulae)
	if cfg.Reattach {
		if _, ok := have[brewToken(pamFormulaReattach)]; !ok {
			out.Formulae = append(out.Formulae, pamFormulaReattach)
		}
	}
	return out
}

// PAMDiffInput is the current PAM filesystem snapshot and resolved module paths
// used to plan sudo_local reconcile actions. Paths are injectable for tests;
// production uses /etc/pam.d/sudo_local and /etc/pam.d/sudo.
type PAMDiffInput struct {
	Config           config.SudoLocalPAM
	SudoLocalPath    string
	SudoPath         string
	SudoLocalContent []byte
	SudoLocalExists  bool
	SudoContent      []byte
	ReattachPath     string
	WatchIDPath      string
}

// BuildPAMPlan computes pam_sudo_local_write / include actions.
// Omitted sudo_local (Configured=false) yields an empty plan.
// enable=false writes a managed stub (marker + disabled comment, no auth lines)
// when the file is missing, empty, or already PourOver-managed — never deletes,
// so `auth include sudo_local` stays safe. Unmanaged non-empty files are left alone.
func BuildPAMPlan(in PAMDiffInput) Plan {
	cfg := in.Config
	if !cfg.Configured {
		return Plan{}
	}

	var actions []Action

	if !cfg.Enable {
		if shouldWriteDisabledStub(in) {
			desired := pam.RenderSudoLocal(cfg, "", "")
			actions = append(actions, Action{
				Type:  ActionPAMSudoLocalWrite,
				Name:  in.SudoLocalPath,
				Value: desired,
			})
		}
		return Plan{Actions: actions}
	}

	desired := pam.RenderSudoLocal(cfg, in.ReattachPath, in.WatchIDPath)
	if !in.SudoLocalExists || !bytes.Equal(in.SudoLocalContent, []byte(desired)) {
		actions = append(actions, Action{
			Type:  ActionPAMSudoLocalWrite,
			Name:  in.SudoLocalPath,
			Value: desired,
		})
	}

	if in.SudoPath != "" && !pam.HasSudoLocalInclude(in.SudoContent) {
		actions = append(actions, Action{
			Type: ActionPAMSudoInclude,
			Name: in.SudoPath,
		})
	}

	return Plan{Actions: actions}
}

func shouldWriteDisabledStub(in PAMDiffInput) bool {
	desired := []byte(pam.RenderSudoLocal(in.Config, "", ""))
	if !in.SudoLocalExists {
		return true
	}
	if pam.IsPourOverManaged(in.SudoLocalContent) || len(bytes.TrimSpace(in.SudoLocalContent)) == 0 {
		return !bytes.Equal(in.SudoLocalContent, desired)
	}
	return false
}

// DefaultPAMSudoLocalPath is the system sudo_local PAM config path.
const DefaultPAMSudoLocalPath = "/etc/pam.d/sudo_local"

// DefaultPAMSudoPath is the system sudo PAM config path.
const DefaultPAMSudoPath = "/etc/pam.d/sudo"
