package config

import (
	"fmt"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func decodeManifest(L *lua.LState, lv lua.LValue) (Manifest, error) {
	if lv == lua.LNil {
		return Manifest{}, fmt.Errorf("config must return a table")
	}
	if lv.Type() != lua.LTTable {
		return Manifest{}, fmt.Errorf("config must return a table, got %s", lv.Type())
	}
	root := lv.(*lua.LTable)

	packages, err := decodePackages(L, root, "packages")
	if err != nil {
		return Manifest{}, err
	}
	files, err := decodeFiles(L, root, "files")
	if err != nil {
		return Manifest{}, err
	}
	policy, err := decodePolicy(L, root, "policy")
	if err != nil {
		return Manifest{}, err
	}
	backup, err := decodeBackup(L, root, "backup")
	if err != nil {
		return Manifest{}, err
	}
	macos, err := decodeMacOS(L, root, "macos")
	if err != nil {
		return Manifest{}, err
	}

	return Manifest{
		Packages: packages,
		Files:    files,
		Policy:   policy,
		Backup:   backup,
		MacOS:    macos,
	}, nil
}

func decodeMacOS(L *lua.LState, root *lua.LTable, key string) (MacOS, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return MacOS{}, err
	}
	if !ok {
		return MacOS{}, nil
	}
	return decodeMacOSTable(L, tbl)
}

// decodeMacOSTable decodes a macos table shaped as { defaults = { … }, security = { … } }.
func decodeMacOSTable(L *lua.LState, tbl *lua.LTable) (MacOS, error) {
	defaults, err := decodeMacOSDefaults(L, tbl)
	if err != nil {
		return MacOS{}, err
	}
	security, err := decodeMacOSSecurity(L, tbl)
	if err != nil {
		return MacOS{}, err
	}
	return MacOS{Defaults: defaults, Security: security}, nil
}

func decodeMacOSDefaults(L *lua.LState, macosTbl *lua.LTable) (MacOSDefaults, error) {
	defaultsTbl, ok, err := fieldTable(L, macosTbl, "defaults")
	if err != nil {
		return MacOSDefaults{}, fmt.Errorf("macos.%w", err)
	}
	if !ok {
		return MacOSDefaults{}, nil
	}

	custom, err := fieldCustomSettings(L, defaultsTbl, "custom")
	if err != nil {
		return MacOSDefaults{}, fmt.Errorf("macos.defaults.%w", err)
	}

	sections := make(map[string]map[string]SettingValue)
	var walkErr error
	defaultsTbl.ForEach(func(k, v lua.LValue) {
		if walkErr != nil {
			return
		}
		if k.Type() != lua.LTString {
			return
		}
		name := k.String()
		if name == "" || name == "custom" {
			return
		}
		if !IsCatalogSection(name) {
			walkErr = fmt.Errorf("macos.defaults.%s: unknown section (see docs/macos-defaults.md; use macos.defaults.custom for arbitrary domains)", name)
			return
		}
		if v.Type() != lua.LTTable {
			walkErr = fmt.Errorf("macos.defaults.%s: expected table, got %s", name, v.Type())
			return
		}
		m, err := decodeSettingMapFromTable(L, v.(*lua.LTable), name)
		if err != nil {
			walkErr = fmt.Errorf("macos.defaults.%w", err)
			return
		}
		if len(m) > 0 {
			sections[name] = m
		}
	})
	if walkErr != nil {
		return MacOSDefaults{}, walkErr
	}

	return MacOSDefaults{
		Sections: sections,
		Custom:   custom,
	}, nil
}

func decodeMacOSSecurity(L *lua.LState, macosTbl *lua.LTable) (MacOSSecurity, error) {
	secTbl, ok, err := fieldTable(L, macosTbl, "security")
	if err != nil {
		return MacOSSecurity{}, fmt.Errorf("macos.%w", err)
	}
	if !ok {
		return MacOSSecurity{}, nil
	}

	pamTbl, ok, err := fieldTable(L, secTbl, "pam")
	if err != nil {
		return MacOSSecurity{}, fmt.Errorf("macos.security.%w", err)
	}
	if !ok {
		return MacOSSecurity{}, nil
	}

	sudoLocal, err := decodeSudoLocalPAM(L, pamTbl)
	if err != nil {
		return MacOSSecurity{}, err
	}
	return MacOSSecurity{PAM: MacOSPAM{SudoLocal: sudoLocal}}, nil
}

func decodeSudoLocalPAM(L *lua.LState, pamTbl *lua.LTable) (SudoLocalPAM, error) {
	tbl, ok, err := fieldTable(L, pamTbl, "sudo_local")
	if err != nil {
		return SudoLocalPAM{}, fmt.Errorf("macos.security.pam.%w", err)
	}
	if !ok {
		return SudoLocalPAM{}, nil
	}

	out := SudoLocalPAM{Configured: true}
	prefix := "macos.security.pam.sudo_local"

	enable, err := optionalBoolDefault(L, tbl, "enable", prefix, true)
	if err != nil {
		return SudoLocalPAM{}, err
	}
	out.Enable = enable

	reattach, err := optionalBool(L, tbl, "reattach", prefix)
	if err != nil {
		return SudoLocalPAM{}, err
	}
	out.Reattach = reattach

	touchID, err := optionalBool(L, tbl, "touch_id_auth", prefix)
	if err != nil {
		return SudoLocalPAM{}, err
	}
	out.TouchIDAuth = touchID

	watchID, err := optionalBool(L, tbl, "watch_id_auth", prefix)
	if err != nil {
		return SudoLocalPAM{}, err
	}
	out.WatchIDAuth = watchID

	return out, nil
}

func optionalBool(L *lua.LState, tbl *lua.LTable, key, prefix string) (bool, error) {
	return optionalBoolDefault(L, tbl, key, prefix, false)
}

func optionalBoolDefault(L *lua.LState, tbl *lua.LTable, key, prefix string, def bool) (bool, error) {
	lv := L.GetField(tbl, key)
	if lv == lua.LNil {
		return def, nil
	}
	if lv.Type() != lua.LTBool {
		return false, fmt.Errorf("%s.%s: expected boolean, got %s", prefix, key, lv.Type())
	}
	return lua.LVAsBool(lv), nil
}

func decodeSettingMapFromTable(L *lua.LState, inner *lua.LTable, key string) (map[string]SettingValue, error) {
	out := make(map[string]SettingValue)
	var walkErr error
	inner.ForEach(func(k, v lua.LValue) {
		if walkErr != nil {
			return
		}
		if k.Type() != lua.LTString {
			walkErr = fmt.Errorf("%s: keys must be strings, got %s", key, k.Type())
			return
		}
		name := k.String()
		if name == "" {
			walkErr = fmt.Errorf("%s: key must not be empty", key)
			return
		}
		sv, err := decodeSettingValue(v, fmt.Sprintf("%s.%s", key, name))
		if err != nil {
			walkErr = err
			return
		}
		out[name] = sv
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func fieldCustomSettings(L *lua.LState, tbl *lua.LTable, key string) (map[string]map[string]SettingValue, error) {
	inner, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	out := make(map[string]map[string]SettingValue)
	var walkErr error
	inner.ForEach(func(k, v lua.LValue) {
		if walkErr != nil {
			return
		}
		if k.Type() != lua.LTString {
			walkErr = fmt.Errorf("%s: domain keys must be strings, got %s", key, k.Type())
			return
		}
		domain := k.String()
		if domain == "" {
			walkErr = fmt.Errorf("%s: domain must not be empty", key)
			return
		}
		if v.Type() != lua.LTTable {
			walkErr = fmt.Errorf("%s[%s]: expected table, got %s", key, domain, v.Type())
			return
		}
		keys := make(map[string]SettingValue)
		v.(*lua.LTable).ForEach(func(kk, vv lua.LValue) {
			if walkErr != nil {
				return
			}
			if kk.Type() != lua.LTString {
				walkErr = fmt.Errorf("%s[%s]: keys must be strings, got %s", key, domain, kk.Type())
				return
			}
			name := kk.String()
			if name == "" {
				walkErr = fmt.Errorf("%s[%s]: key must not be empty", key, domain)
				return
			}
			sv, err := decodeSettingValue(vv, fmt.Sprintf("%s[%s].%s", key, domain, name))
			if err != nil {
				walkErr = err
				return
			}
			keys[name] = sv
		})
		if walkErr != nil {
			return
		}
		if len(keys) > 0 {
			out[domain] = keys
		}
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func decodeSettingValue(v lua.LValue, field string) (SettingValue, error) {
	switch v.Type() {
	case lua.LTBool:
		return SettingValue{Kind: SettingBool, Bool: lua.LVAsBool(v)}, nil
	case lua.LTNumber:
		n := float64(v.(lua.LNumber))
		if n == float64(int64(n)) {
			return SettingValue{Kind: SettingInt, Int: int64(n)}, nil
		}
		return SettingValue{Kind: SettingFloat, Float: n}, nil
	case lua.LTString:
		return SettingValue{Kind: SettingString, String: UnescapeLuaUnicode(v.String())}, nil
	case lua.LTTable:
		paths, err := decodeStringArrayTable(v.(*lua.LTable), field)
		if err != nil {
			return SettingValue{}, err
		}
		for i := range paths {
			paths[i] = UnescapeLuaUnicode(paths[i])
		}
		return SettingValue{Kind: SettingArray, Array: paths}, nil
	default:
		return SettingValue{}, fmt.Errorf("%s: expected bool, number, string, or array, got %s", field, v.Type())
	}
}

func decodeStringArrayTable(tbl *lua.LTable, field string) ([]string, error) {
	var paths []string
	var walkErr error
	tbl.ForEach(func(k, v lua.LValue) {
		if walkErr != nil {
			return
		}
		if k.Type() != lua.LTNumber {
			walkErr = fmt.Errorf("%s: array keys must be numbers, got %s", field, k.Type())
			return
		}
		if v.Type() != lua.LTString {
			walkErr = fmt.Errorf("%s[%v]: expected string path, got %s", field, k, v.Type())
			return
		}
		s := strings.TrimSpace(v.String())
		if s == "" {
			walkErr = fmt.Errorf("%s[%v]: path must not be empty", field, k)
			return
		}
		paths = append(paths, s)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return paths, nil
}

func decodePackages(L *lua.LState, root *lua.LTable, key string) (Packages, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Packages{}, err
	}
	if !ok {
		return Packages{}, nil
	}

	taps, err := fieldTapSpecs(L, tbl, "taps")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	formulae, err := fieldStringSlice(L, tbl, "formulae")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	casks, err := fieldStringSlice(L, tbl, "casks")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	mas, masConfigured, err := fieldMasApps(L, tbl, "mas")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	return Packages{
		Taps:          taps,
		Formulae:      formulae,
		Casks:         casks,
		Mas:           mas,
		MasConfigured: masConfigured,
	}, nil
}

// fieldMasApps decodes packages.mas as a string-key → number-id map.
// Omitted key → configured=false; present (including {}) → configured=true.
func fieldMasApps(L *lua.LState, tbl *lua.LTable, key string) ([]MasApp, bool, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	var out []MasApp
	var walkErr error
	arr.ForEach(func(k, v lua.LValue) {
		if walkErr != nil {
			return
		}
		if k.Type() != lua.LTString {
			walkErr = fmt.Errorf("%s: keys must be strings (app names), got %s", key, k.Type())
			return
		}
		name := k.String()
		if v.Type() != lua.LTNumber {
			walkErr = fmt.Errorf("%s[%q]: expected number id, got %s", key, name, v.Type())
			return
		}
		n := float64(v.(lua.LNumber))
		id := int64(n)
		if n != float64(id) {
			walkErr = fmt.Errorf("%s[%q]: id must be an integer, got %v", key, name, n)
			return
		}
		out = append(out, MasApp{Name: name, ID: id})
	})
	if walkErr != nil {
		return nil, true, walkErr
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, true, nil
}

func decodeFiles(L *lua.LState, root *lua.LTable, key string) (Files, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Files{}, err
	}
	if !ok {
		return Files{}, nil
	}

	links, err := fieldFileLinks(L, tbl, "links")
	if err != nil {
		return Files{}, fmt.Errorf("files.%w", err)
	}
	managed, err := fieldManagedFiles(L, tbl, "managed")
	if err != nil {
		return Files{}, fmt.Errorf("files.%w", err)
	}
	templates, err := fieldTemplateFiles(L, tbl, "templates")
	if err != nil {
		return Files{}, fmt.Errorf("files.%w", err)
	}
	unlink, err := fieldStringSlice(L, tbl, "unlink")
	if err != nil {
		return Files{}, fmt.Errorf("files.%w", err)
	}
	return Files{Links: links, Managed: managed, Templates: templates, Unlink: unlink}, nil
}

func decodePolicy(L *lua.LState, root *lua.LTable, key string) (Policy, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Policy{}, err
	}
	if !ok {
		return Policy{UninstallMode: UninstallModeSafe, FilesMode: FilesModeSafe}, nil
	}

	p := Policy{UninstallMode: UninstallModeSafe, FilesMode: FilesModeSafe}

	modeLV := L.GetField(tbl, "uninstall_mode")
	if modeLV != lua.LNil {
		if modeLV.Type() != lua.LTString {
			return Policy{}, fmt.Errorf("policy.uninstall_mode: expected string, got %s", modeLV.Type())
		}
		p.UninstallMode = UninstallMode(modeLV.String())
	}

	replaceLV := L.GetField(tbl, "file_replace")
	if replaceLV != lua.LNil {
		if replaceLV.Type() != lua.LTString {
			return Policy{}, fmt.Errorf("policy.file_replace: expected string, got %s", replaceLV.Type())
		}
		p.FileReplace = FileReplaceMode(replaceLV.String())
	}

	filesModeLV := L.GetField(tbl, "files_mode")
	if filesModeLV != lua.LNil {
		if filesModeLV.Type() != lua.LTString {
			return Policy{}, fmt.Errorf("policy.files_mode: expected string, got %s", filesModeLV.Type())
		}
		p.FilesMode = FilesMode(filesModeLV.String())
	}
	return p, nil
}

func decodeBackup(L *lua.LState, root *lua.LTable, key string) (Backup, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Backup{}, err
	}
	if !ok {
		return Backup{}, nil
	}

	icloud, err := decodeICloudBackup(L, tbl)
	if err != nil {
		return Backup{}, err
	}
	git, err := decodeGitBackup(L, tbl)
	if err != nil {
		return Backup{}, err
	}
	return Backup{ICloud: icloud, Git: git}, nil
}

func decodeICloudBackup(L *lua.LState, backupTbl *lua.LTable) (ICloudBackup, error) {
	icloudTbl, ok, err := fieldTable(L, backupTbl, "icloud")
	if err != nil {
		return ICloudBackup{}, fmt.Errorf("backup.%w", err)
	}
	if !ok {
		return ICloudBackup{}, nil
	}

	enabled := false
	enabledLV := L.GetField(icloudTbl, "enabled")
	if enabledLV != lua.LNil {
		if enabledLV.Type() != lua.LTBool {
			return ICloudBackup{}, fmt.Errorf("backup.icloud.enabled: expected boolean, got %s", enabledLV.Type())
		}
		enabled = lua.LVAsBool(enabledLV)
	}

	path := ""
	pathLV := L.GetField(icloudTbl, "path")
	if pathLV != lua.LNil {
		if pathLV.Type() != lua.LTString {
			return ICloudBackup{}, fmt.Errorf("backup.icloud.path: expected string, got %s", pathLV.Type())
		}
		path = pathLV.String()
	}

	return ICloudBackup{Enabled: enabled, Path: path}, nil
}

func decodeGitBackup(L *lua.LState, backupTbl *lua.LTable) (GitBackup, error) {
	gitTbl, ok, err := fieldTable(L, backupTbl, "git")
	if err != nil {
		return GitBackup{}, fmt.Errorf("backup.%w", err)
	}
	if !ok {
		return GitBackup{}, nil
	}

	out := GitBackup{AutoPush: true} // default when table present but auto_push omitted
	enabledLV := L.GetField(gitTbl, "enabled")
	if enabledLV != lua.LNil {
		if enabledLV.Type() != lua.LTBool {
			return GitBackup{}, fmt.Errorf("backup.git.enabled: expected boolean, got %s", enabledLV.Type())
		}
		out.Enabled = lua.LVAsBool(enabledLV)
	}

	remoteLV := L.GetField(gitTbl, "remote")
	if remoteLV != lua.LNil {
		if remoteLV.Type() != lua.LTString {
			return GitBackup{}, fmt.Errorf("backup.git.remote: expected string, got %s", remoteLV.Type())
		}
		out.Remote = remoteLV.String()
	}

	autoLV := L.GetField(gitTbl, "auto_push")
	if autoLV != lua.LNil {
		if autoLV.Type() != lua.LTBool {
			return GitBackup{}, fmt.Errorf("backup.git.auto_push: expected boolean, got %s", autoLV.Type())
		}
		out.AutoPush = lua.LVAsBool(autoLV)
	}

	branchLV := L.GetField(gitTbl, "branch")
	if branchLV != lua.LNil {
		if branchLV.Type() != lua.LTString {
			return GitBackup{}, fmt.Errorf("backup.git.branch: expected string, got %s", branchLV.Type())
		}
		out.Branch = branchLV.String()
	}

	return out, nil
}

func fieldFileLinks(L *lua.LState, tbl *lua.LTable, key string) ([]FileLink, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var links []FileLink
	for i := 1; i <= arr.Len(); i++ {
		entry := L.RawGetInt(arr, i)
		if entry.Type() != lua.LTTable {
			return nil, fmt.Errorf("%s[%d]: expected table, got %s", key, i, entry.Type())
		}
		link, err := decodeFileLink(L, entry.(*lua.LTable), key, i)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func fieldManagedFiles(L *lua.LState, tbl *lua.LTable, key string) ([]ManagedFile, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var files []ManagedFile
	for i := 1; i <= arr.Len(); i++ {
		entry := L.RawGetInt(arr, i)
		if entry.Type() != lua.LTTable {
			return nil, fmt.Errorf("%s[%d]: expected table, got %s", key, i, entry.Type())
		}
		file, err := decodeManagedFile(L, entry.(*lua.LTable), key, i)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func fieldTemplateFiles(L *lua.LState, tbl *lua.LTable, key string) ([]TemplateFile, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var files []TemplateFile
	for i := 1; i <= arr.Len(); i++ {
		entry := L.RawGetInt(arr, i)
		if entry.Type() != lua.LTTable {
			return nil, fmt.Errorf("%s[%d]: expected table, got %s", key, i, entry.Type())
		}
		file, err := decodeTemplateFile(L, entry.(*lua.LTable), key, i)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func decodeFileLink(L *lua.LState, tbl *lua.LTable, field string, index int) (FileLink, error) {
	prefix := fmt.Sprintf("%s[%d]", field, index)
	source, err := requiredString(L, tbl, "source", prefix)
	if err != nil {
		return FileLink{}, err
	}
	target, err := requiredString(L, tbl, "target", prefix)
	if err != nil {
		return FileLink{}, err
	}
	return FileLink{Source: source, Target: target}, nil
}

func decodeManagedFile(L *lua.LState, tbl *lua.LTable, field string, index int) (ManagedFile, error) {
	prefix := fmt.Sprintf("%s[%d]", field, index)
	source, err := requiredString(L, tbl, "source", prefix)
	if err != nil {
		return ManagedFile{}, err
	}
	target, err := requiredString(L, tbl, "target", prefix)
	if err != nil {
		return ManagedFile{}, err
	}
	return ManagedFile{Source: source, Target: target}, nil
}

func decodeTemplateFile(L *lua.LState, tbl *lua.LTable, field string, index int) (TemplateFile, error) {
	prefix := fmt.Sprintf("%s[%d]", field, index)
	source, err := requiredString(L, tbl, "source", prefix)
	if err != nil {
		return TemplateFile{}, err
	}
	target, err := requiredString(L, tbl, "target", prefix)
	if err != nil {
		return TemplateFile{}, err
	}
	return TemplateFile{Source: source, Target: target}, nil
}

func fieldStringSlice(L *lua.LState, tbl *lua.LTable, key string) ([]string, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var out []string
	for i := 1; i <= arr.Len(); i++ {
		v := L.RawGetInt(arr, i)
		if v.Type() != lua.LTString {
			return nil, fmt.Errorf("%s[%d]: expected string, got %s", key, i, v.Type())
		}
		out = append(out, v.String())
	}
	return out, nil
}

func fieldTapSpecs(L *lua.LState, tbl *lua.LTable, key string) ([]TapSpec, error) {
	arr, ok, err := fieldTable(L, tbl, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var out []TapSpec
	for i := 1; i <= arr.Len(); i++ {
		v := L.RawGetInt(arr, i)
		switch v.Type() {
		case lua.LTString:
			out = append(out, TapSpec{Name: v.String(), Trusted: true})
		case lua.LTTable:
			tap, err := decodeTapSpec(L, v.(*lua.LTable), key, i)
			if err != nil {
				return nil, err
			}
			out = append(out, tap)
		default:
			return nil, fmt.Errorf("%s[%d]: expected string or table, got %s", key, i, v.Type())
		}
	}
	return out, nil
}

func decodeTapSpec(L *lua.LState, tbl *lua.LTable, field string, index int) (TapSpec, error) {
	prefix := fmt.Sprintf("%s[%d]", field, index)
	name, err := requiredString(L, tbl, "name", prefix)
	if err != nil {
		return TapSpec{}, err
	}
	trusted := true
	trustedLV := L.GetField(tbl, "trusted")
	if trustedLV != lua.LNil {
		if trustedLV.Type() != lua.LTBool {
			return TapSpec{}, fmt.Errorf("%s.trusted: expected boolean, got %s", prefix, trustedLV.Type())
		}
		trusted = lua.LVAsBool(trustedLV)
	}
	url := ""
	urlLV := L.GetField(tbl, "url")
	if urlLV != lua.LNil {
		if urlLV.Type() != lua.LTString {
			return TapSpec{}, fmt.Errorf("%s.url: expected string, got %s", prefix, urlLV.Type())
		}
		url = strings.TrimSpace(urlLV.String())
	}
	return TapSpec{Name: name, Trusted: trusted, URL: url}, nil
}

func fieldTable(L *lua.LState, tbl *lua.LTable, key string) (*lua.LTable, bool, error) {
	lv := L.GetField(tbl, key)
	if lv == lua.LNil {
		return nil, false, nil
	}
	if lv.Type() != lua.LTTable {
		return nil, false, fmt.Errorf("%s: expected table, got %s", key, lv.Type())
	}
	return lv.(*lua.LTable), true, nil
}

func requiredString(L *lua.LState, tbl *lua.LTable, key, prefix string) (string, error) {
	lv := L.GetField(tbl, key)
	if lv == lua.LNil {
		return "", fmt.Errorf("%s.%s: required string", prefix, key)
	}
	if lv.Type() != lua.LTString {
		return "", fmt.Errorf("%s.%s: expected string, got %s", prefix, key, lv.Type())
	}
	return lv.String(), nil
}
