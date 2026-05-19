package config

import (
	"fmt"
	"strings"
)

// Validate checks a decoded manifest for semantic errors.
func Validate(m *Manifest) error {
	var errs []error

	errs = append(errs, validatePolicy(m.Policy)...)

	for i, name := range m.Packages.Formulae {
		if err := validatePackageName(name, fmt.Sprintf("packages.formulae[%d]", i+1)); err != nil {
			errs = append(errs, err)
		}
	}
	for i, name := range m.Packages.Casks {
		if err := validatePackageName(name, fmt.Sprintf("packages.casks[%d]", i+1)); err != nil {
			errs = append(errs, err)
		}
	}

	for i, link := range m.Files.Links {
		prefix := fmt.Sprintf("files.links[%d]", i+1)
		if err := validatePathField(link.Source, prefix+".source"); err != nil {
			errs = append(errs, err)
		}
		if err := validatePathField(link.Target, prefix+".target"); err != nil {
			errs = append(errs, err)
		}
	}
	for i, file := range m.Files.Managed {
		prefix := fmt.Sprintf("files.managed[%d]", i+1)
		if err := validatePathField(file.Source, prefix+".source"); err != nil {
			errs = append(errs, err)
		}
		if err := validatePathField(file.Target, prefix+".target"); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("invalid config: %s", strings.Join(msgs, "; "))
}

func validatePolicy(p Policy) []error {
	if p.UninstallMode == "" {
		return nil
	}
	if isValidUninstallMode(p.UninstallMode) {
		return nil
	}
	return []error{fmt.Errorf("policy.uninstall_mode: unknown value %q (want safe, strict, or non_destructive)", p.UninstallMode)}
}

func isValidUninstallMode(m UninstallMode) bool {
	switch m {
	case UninstallModeSafe, UninstallModeStrict, UninstallModeNonDestructive:
		return true
	default:
		return false
	}
}

func validatePackageName(name, field string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s: must not be empty", field)
	}
	return nil
}

func validatePathField(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s: must not be empty", field)
	}
	return nil
}
