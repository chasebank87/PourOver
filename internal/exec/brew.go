package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// AddTap runs `brew tap <name> [url]` then, when trusted is true, `brew trust --tap <name>`
// for non-official taps (Homebrew 6+ requires explicit trust before loading third-party tap code).
// url is optional; when set it selects a custom clone remote (needed when the repo is not
// named homebrew-<tap>).
func AddTap(ctx context.Context, runner discovery.Runner, name, url string, trusted bool) error {
	args := []string{"tap", name}
	if url != "" {
		args = append(args, url)
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		if url != "" {
			return fmt.Errorf("brew tap %s %s: %w", name, url, err)
		}
		return fmt.Errorf("brew tap %s: %w", name, err)
	}
	if trusted && discovery.NeedsExplicitTrust(name) {
		if err := TrustTap(ctx, runner, name); err != nil {
			return err
		}
	}
	return nil
}

// TrustTap runs `brew trust --tap <name>`.
func TrustTap(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "trust", "--tap", name); err != nil {
		return fmt.Errorf("brew trust --tap %s: %w", name, err)
	}
	return nil
}

// UpdateBrew runs `brew update` so newly tapped formulae/casks are visible to install.
func UpdateBrew(ctx context.Context, runner discovery.Runner) error {
	if _, err := runner.Run(ctx, "update"); err != nil {
		return fmt.Errorf("brew update: %w", err)
	}
	return nil
}

// InstallFormula runs `brew install <name>`.
func InstallFormula(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "install", name); err != nil {
		return fmt.Errorf("brew install %s: %w", name, err)
	}
	return nil
}

// InstallCask runs `brew install --cask <name>`.
func InstallCask(ctx context.Context, runner discovery.Runner, name string) error {
	return InstallCasks(ctx, runner, []string{name})
}

// InstallCasks runs `brew install --cask` for all names in one invocation so
// Homebrew only elevates once for the batch.
func InstallCasks(ctx context.Context, runner discovery.Runner, names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"install", "--cask"}, names...)
	if _, err := runner.Run(ctx, args...); err != nil {
		return err
	}
	return nil
}

// ApplyTapAdds runs brew tap (and trust) for each tap_add action, and brew trust
// for each tap_trust action, in plan order. After any successful tap_add it runs
// `brew update` once so packages from new taps can be installed.
// Failures are collected so later taps still run. Returns successes and joined errors.
func ApplyTapAdds(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	added := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionTapAdd:
			report(progress, a)
			if err := AddTap(ctx, runner, a.Name, a.URL, a.Trusted); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
			added++
		case plan.ActionTapTrust:
			report(progress, a)
			if err := TrustTap(ctx, runner, a.Name); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
		}
	}
	if added > 0 {
		if progress != nil {
			progress("brew update")
		}
		if err := UpdateBrew(ctx, runner); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
		}
	}
	return n, errors.Join(errs...)
}

// ApplyFormulaInstalls runs brew install for each formula_install action in plan order.
// A failure for one formula is recorded and the remaining installs still run (nix-darwin-style).
// Returns how many installs succeeded; error is errors.Join of any failures.
func ApplyFormulaInstalls(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionFormulaInstall {
			continue
		}
		report(progress, a)
		if err := InstallFormula(ctx, runner, a.Name); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// CaskInstallChunkSize is how many casks to install per brew invocation.
// Chunking lets a failed/killed batch leave earlier chunks installed so the
// next apply rediscovers only leftovers.
const CaskInstallChunkSize = 8

// ApplyCaskInstalls runs brew install --cask for cask_install actions in chunks
// of CaskInstallChunkSize. Admin credentials are cached once with sudo -v.
// A failed chunk is recorded and later chunks still run (formulae-style).
// beforeAuth parks fancy UI before that prompt. Returns successes and joined errors.
func ApplyCaskInstalls(ctx context.Context, runner discovery.Runner, p plan.Plan, beforeAuth func(), progress Progress) (int, error) {
	var casks []plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionCaskInstall {
			casks = append(casks, a)
		}
	}
	if len(casks) == 0 {
		return 0, nil
	}

	if err := EnsureSudo(ctx, beforeAuth); err != nil {
		return 0, err
	}

	n := 0
	var errs []error
	for i := 0; i < len(casks); i += CaskInstallChunkSize {
		end := i + CaskInstallChunkSize
		if end > len(casks) {
			end = len(casks)
		}
		chunk := casks[i:end]
		for _, a := range chunk {
			report(progress, a)
		}
		names := make([]string, len(chunk))
		for j, a := range chunk {
			names[j] = a.Name
		}
		if err := InstallCasks(ctx, runner, names); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n += len(chunk)
	}
	return n, errors.Join(errs...)
}

// UnsupportedApplyActions returns plan actions that apply does not run yet.
// Advisory actions like cask_rename are excluded (handled separately).
func UnsupportedApplyActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionTapAdd, plan.ActionTapTrust, plan.ActionTapRemove,
			plan.ActionFormulaInstall, plan.ActionCaskInstall,
			plan.ActionFormulaRemove, plan.ActionCaskRemove,
			plan.ActionMasInstall, plan.ActionMasRemove,
			plan.ActionLinkCreate, plan.ActionLinkUpdate, plan.ActionLinkReplace,
			plan.ActionManagedCopy, plan.ActionTemplateWrite, plan.ActionFileUnlink, plan.ActionFilePrune,
			plan.ActionDefaultsWrite,
			plan.ActionPAMSudoLocalWrite, plan.ActionPAMSudoLocalRemove, plan.ActionPAMSudoInclude,
			plan.ActionCaskRename:
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}

// CaskRenameActions returns advisory cask_rename actions from the plan.
func CaskRenameActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionCaskRename {
			out = append(out, a)
		}
	}
	return out
}
