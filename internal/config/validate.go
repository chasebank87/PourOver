package config

import (
	"fmt"
	"strings"
)

// Validate checks a decoded manifest for semantic errors.
func Validate(m *Manifest) error {
	var errs []error

	errs = append(errs, validatePolicy(m.Policy)...)

	for i, tap := range m.Packages.Taps {
		if err := validatePackageName(tap.Name, fmt.Sprintf("packages.taps[%d]", i+1)); err != nil {
			errs = append(errs, err)
		}
	}
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
	errs = append(errs, validateMasApps(m.Packages.Mas)...)

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
	for i, file := range m.Files.Templates {
		prefix := fmt.Sprintf("files.templates[%d]", i+1)
		if err := validatePathField(file.Source, prefix+".source"); err != nil {
			errs = append(errs, err)
		}
		if err := validatePathField(file.Target, prefix+".target"); err != nil {
			errs = append(errs, err)
		}
	}
	for i, path := range m.Files.Unlink {
		if err := validatePathField(path, fmt.Sprintf("files.unlink[%d]", i+1)); err != nil {
			errs = append(errs, err)
		}
	}

	errs = append(errs, validateMacOS(m.MacOS)...)

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
	var errs []error
	if p.UninstallMode != "" && !isValidUninstallMode(p.UninstallMode) {
		errs = append(errs, fmt.Errorf("policy.uninstall_mode: unknown value %q (want safe, strict, or non_destructive)", p.UninstallMode))
	}
	if p.FileReplace != "" && !isValidFileReplaceMode(p.FileReplace) {
		errs = append(errs, fmt.Errorf("policy.file_replace: unknown value %q (want error, backup, or force)", p.FileReplace))
	}
	if p.FilesMode != "" && !isValidFilesMode(p.FilesMode) {
		errs = append(errs, fmt.Errorf("policy.files_mode: unknown value %q (want safe, strict, or non_destructive)", p.FilesMode))
	}
	return errs
}

func isValidUninstallMode(m UninstallMode) bool {
	switch m {
	case UninstallModeSafe, UninstallModeStrict, UninstallModeNonDestructive:
		return true
	default:
		return false
	}
}

func isValidFileReplaceMode(m FileReplaceMode) bool {
	switch m {
	case FileReplaceError, FileReplaceBackup, FileReplaceMode("force"):
		return true
	default:
		return false
	}
}

func isValidFilesMode(m FilesMode) bool {
	switch m {
	case FilesModeSafe, FilesModeStrict, FilesModeNonDestructive:
		return true
	default:
		return false
	}
}

func validatePackageName(name, field string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%s: must not be empty", field)
	}
	if trimmed != name {
		return fmt.Errorf("%s: must not have leading/trailing whitespace (got %q)", field, name)
	}
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			return fmt.Errorf("%s: %q must be a lowercase Homebrew token (use %q)", field, name, strings.ToLower(name))
		}
	}
	return nil
}

func validateMasApps(apps []MasApp) []error {
	var errs []error
	seenIDs := make(map[int64]string, len(apps))
	seenNames := make(map[string]int64, len(apps))
	for i, app := range apps {
		field := fmt.Sprintf("packages.mas[%d]", i+1)
		name := strings.TrimSpace(app.Name)
		if name == "" {
			errs = append(errs, fmt.Errorf("%s: name must not be empty", field))
		} else if name != app.Name {
			errs = append(errs, fmt.Errorf("%s: name must not have leading/trailing whitespace (got %q)", field, app.Name))
		}
		if app.ID <= 0 {
			errs = append(errs, fmt.Errorf("%s: id must be a positive integer (got %d)", field, app.ID))
		}
		if prev, ok := seenIDs[app.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate id %d (also used by %q)", field, app.ID, prev))
		} else if app.ID > 0 {
			seenIDs[app.ID] = app.Name
		}
		if app.Name != "" {
			if prevID, ok := seenNames[app.Name]; ok {
				errs = append(errs, fmt.Errorf("%s: duplicate name %q (also id %d)", field, app.Name, prevID))
			} else {
				seenNames[app.Name] = app.ID
			}
		}
	}
	return errs
}

func validatePathField(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s: must not be empty", field)
	}
	return nil
}

func validateMacOS(m MacOS) []error {
	var errs []error
	d := m.Defaults
	for section, keys := range d.Sections {
		errs = append(errs, validateCuratedSettings("macos.defaults."+section, section, keys)...)
	}
	for domain, keys := range d.Custom {
		if strings.TrimSpace(domain) == "" {
			errs = append(errs, fmt.Errorf("macos.defaults.custom: domain must not be empty"))
			continue
		}
		for key, val := range keys {
			prefix := fmt.Sprintf("macos.defaults.custom[%s].%s", domain, key)
			if strings.TrimSpace(key) == "" {
				errs = append(errs, fmt.Errorf("%s: key must not be empty", prefix))
			}
			if err := validateSettingValue(val, prefix); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

func validateCuratedSettings(prefix, section string, keys map[string]SettingValue) []error {
	var errs []error
	for key, val := range keys {
		field := prefix + "." + key
		if strings.TrimSpace(key) == "" {
			errs = append(errs, fmt.Errorf("%s: key must not be empty", prefix))
			continue
		}
		if !IsCuratedKey(section, key) {
			errs = append(errs, fmt.Errorf("%s: unknown curated key %q (see docs/macos-defaults.md; use macos.defaults.custom for arbitrary domains)", field, key))
			continue
		}
		if err := validateSettingValue(val, field); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateSettingValue(v SettingValue, field string) error {
	switch v.Kind {
	case SettingBool, SettingInt, SettingFloat, SettingString:
		return nil
	case SettingArray:
		for i, p := range v.Array {
			if strings.TrimSpace(p) == "" {
				return fmt.Errorf("%s[%d]: path must not be empty", field, i+1)
			}
		}
		return nil
	case "":
		return fmt.Errorf("%s: missing value kind", field)
	default:
		return fmt.Errorf("%s: unsupported value kind %q", field, v.Kind)
	}
}
