package plan

import (
	"bytes"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/pam"
)

const (
	pamFormulaReattach = "pam-reattach"
	pamFormulaWatchID  = "pam-watchid"
)

// ExpandPAMFormulae appends pam-reattach / pam-watchid to desired formulae when
// the corresponding sudo_local flags are enabled and the formula is not already listed.
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
			have[brewToken(pamFormulaReattach)] = struct{}{}
		}
	}
	if cfg.WatchIDAuth {
		if _, ok := have[brewToken(pamFormulaWatchID)]; !ok {
			out.Formulae = append(out.Formulae, pamFormulaWatchID)
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

// BuildPAMPlan computes pam_sudo_local_write / remove / include actions.
// Omitted sudo_local (Configured=false) yields an empty plan.
// enable=false removes only a PourOver-managed file (marker) or one whose
// content already matches the empty disabled render.
func BuildPAMPlan(in PAMDiffInput) Plan {
	cfg := in.Config
	if !cfg.Configured {
		return Plan{}
	}

	var actions []Action

	if !cfg.Enable {
		if in.SudoLocalExists && shouldRemoveSudoLocal(in.SudoLocalContent) {
			actions = append(actions, Action{
				Type: ActionPAMSudoLocalRemove,
				Name: in.SudoLocalPath,
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

func shouldRemoveSudoLocal(content []byte) bool {
	if pam.IsPourOverManaged(content) {
		return true
	}
	// Content matches disabled render (empty): treat as safe to remove.
	return len(bytes.TrimSpace(content)) == 0
}

// DefaultPAMSudoLocalPath is the system sudo_local PAM config path.
const DefaultPAMSudoLocalPath = "/etc/pam.d/sudo_local"

// DefaultPAMSudoPath is the system sudo PAM config path.
const DefaultPAMSudoPath = "/etc/pam.d/sudo"
