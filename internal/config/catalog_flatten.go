package config

import "sort"

// CatalogDesiredSettings flattens the embedded macos catalog into one DesiredSetting
// per (domain, key), including AlsoDomains, for import/snapshot.
func CatalogDesiredSettings() []DesiredSetting {
	sections := Catalog()
	var out []DesiredSetting
	for _, s := range sections {
		keys := make([]CatalogKey, len(s.Keys))
		copy(keys, s.Keys)
		sort.Slice(keys, func(i, j int) bool { return keys[i].Key < keys[j].Key })
		domains := append([]string{s.Domain}, s.AlsoDomains...)
		for _, k := range keys {
			for _, domain := range domains {
				out = append(out, DesiredSetting{
					Domain:  domain,
					Key:     k.Key,
					Value:   SettingValue{Kind: SettingKind(k.Kind)},
					Section: s.Name,
					Scope:   s.Scope,
				})
			}
		}
	}
	return out
}
