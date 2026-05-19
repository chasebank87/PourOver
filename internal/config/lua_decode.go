package config

import (
	"fmt"

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

	return Manifest{
		Packages: packages,
		Files:    files,
		Policy:   policy,
		Backup:   backup,
	}, nil
}

func decodePackages(L *lua.LState, root *lua.LTable, key string) (Packages, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Packages{}, err
	}
	if !ok {
		return Packages{}, nil
	}

	formulae, err := fieldStringSlice(L, tbl, "formulae")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	casks, err := fieldStringSlice(L, tbl, "casks")
	if err != nil {
		return Packages{}, fmt.Errorf("packages.%w", err)
	}
	return Packages{Formulae: formulae, Casks: casks}, nil
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
	return Files{Links: links, Managed: managed}, nil
}

func decodePolicy(L *lua.LState, root *lua.LTable, key string) (Policy, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Policy{}, err
	}
	if !ok {
		return Policy{UninstallMode: UninstallModeSafe}, nil
	}

	modeLV := L.GetField(tbl, "uninstall_mode")
	if modeLV == lua.LNil {
		return Policy{UninstallMode: UninstallModeSafe}, nil
	}
	if modeLV.Type() != lua.LTString {
		return Policy{}, fmt.Errorf("policy.uninstall_mode: expected string, got %s", modeLV.Type())
	}
	return Policy{UninstallMode: UninstallMode(modeLV.String())}, nil
}

func decodeBackup(L *lua.LState, root *lua.LTable, key string) (Backup, error) {
	tbl, ok, err := fieldTable(L, root, key)
	if err != nil {
		return Backup{}, err
	}
	if !ok {
		return Backup{}, nil
	}

	icloudTbl, ok, err := fieldTable(L, tbl, "icloud")
	if err != nil {
		return Backup{}, fmt.Errorf("backup.%w", err)
	}
	if !ok {
		return Backup{}, nil
	}

	enabled := false
	enabledLV := L.GetField(icloudTbl, "enabled")
	if enabledLV != lua.LNil {
		if enabledLV.Type() != lua.LTBool {
			return Backup{}, fmt.Errorf("backup.icloud.enabled: expected boolean, got %s", enabledLV.Type())
		}
		enabled = lua.LVAsBool(enabledLV)
	}

	path := ""
	pathLV := L.GetField(icloudTbl, "path")
	if pathLV != lua.LNil {
		if pathLV.Type() != lua.LTString {
			return Backup{}, fmt.Errorf("backup.icloud.path: expected string, got %s", pathLV.Type())
		}
		path = pathLV.String()
	}

	return Backup{ICloud: ICloudBackup{Enabled: enabled, Path: path}}, nil
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
