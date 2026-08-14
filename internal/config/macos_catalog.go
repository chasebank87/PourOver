package config

import (
	_ "embed"
	"fmt"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// Apple domains for common curated sections.
const (
	DomainDock           = "com.apple.dock"
	DomainFinder         = "com.apple.finder"
	DomainNSGlobalDomain = "-g"
)

//go:embed macos_catalog.yaml
var catalogYAML []byte

// CatalogSection is one named Lua table under macos.defaults.
type CatalogSection struct {
	Name        string       `yaml:"name"`
	Domain      string       `yaml:"domain"`
	AlsoDomains []string     `yaml:"also_domains,omitempty"` // extra Apple domains (e.g. Bluetooth trackpad)
	Scope       string       `yaml:"scope"`                  // user | system | byhost
	Killall     string       `yaml:"killall,omitempty"`
	MyNixOS     string       `yaml:"mynixos"`
	Keys        []CatalogKey `yaml:"keys"`
}

// CatalogKey is one defaults key in a section.
type CatalogKey struct {
	Key         string `yaml:"key"`
	Kind        string `yaml:"kind"`
	Description string `yaml:"description,omitempty"`
}

type catalogFile struct {
	Sections []CatalogSection `yaml:"sections"`
}

var (
	catalogOnce     sync.Once
	catalogErr      error
	catalogSections []CatalogSection
	sectionByName   map[string]CatalogSection
	keysBySection   map[string]map[string]CatalogKey
	sectionByDomain map[string][]CatalogSection
)

func loadCatalog() {
	catalogOnce.Do(func() {
		var file catalogFile
		if err := yaml.Unmarshal(catalogYAML, &file); err != nil {
			catalogErr = fmt.Errorf("macos catalog: %w", err)
			return
		}
		sectionByName = make(map[string]CatalogSection, len(file.Sections))
		keysBySection = make(map[string]map[string]CatalogKey, len(file.Sections))
		sectionByDomain = make(map[string][]CatalogSection)
		for _, s := range file.Sections {
			sectionByName[s.Name] = s
			km := make(map[string]CatalogKey, len(s.Keys))
			for _, k := range s.Keys {
				km[k.Key] = k
			}
			keysBySection[s.Name] = km
			sectionByDomain[s.Domain] = append(sectionByDomain[s.Domain], s)
		}
		catalogSections = file.Sections
	})
}

// Catalog returns the embedded nix-darwin defaults catalog.
func Catalog() []CatalogSection {
	loadCatalog()
	return catalogSections
}

// CatalogLoadError is non-nil if the embedded YAML failed to parse.
func CatalogLoadError() error {
	loadCatalog()
	return catalogErr
}

// IsCatalogSection reports whether name is a first-class macos.defaults table.
func IsCatalogSection(name string) bool {
	loadCatalog()
	_, ok := sectionByName[name]
	return ok
}

// IsCuratedKey reports whether key is allowed under a curated section.
func IsCuratedKey(section, key string) bool {
	loadCatalog()
	keys, ok := keysBySection[section]
	if !ok {
		return false
	}
	_, ok = keys[key]
	return ok
}

// DomainForSection returns the Apple defaults domain for a curated section.
func DomainForSection(section string) (string, bool) {
	loadCatalog()
	s, ok := sectionByName[section]
	if !ok {
		return "", false
	}
	return s.Domain, true
}

// KillallForDomain returns UI process names to restart after writing domain.
func KillallForDomain(domain string) []string {
	loadCatalog()
	seen := map[string]struct{}{}
	var out []string
	for _, s := range sectionByDomain[domain] {
		if s.Killall == "" {
			continue
		}
		if _, ok := seen[s.Killall]; ok {
			continue
		}
		seen[s.Killall] = struct{}{}
		out = append(out, s.Killall)
	}
	return out
}

// DesiredSetting is one flattened preference to reconcile.
type DesiredSetting struct {
	Domain  string
	Key     string
	Value   SettingValue
	Section string // catalog section name or "custom"
	Scope   string // user | system | byhost | custom
}

// FlattenDefaults expands curated + custom maps into a deterministic list.
func FlattenDefaults(d MacOSDefaults) []DesiredSetting {
	loadCatalog()
	var out []DesiredSetting
	names := make([]string, 0, len(d.Sections))
	for n := range d.Sections {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, section := range names {
		meta, ok := sectionByName[section]
		if !ok {
			continue
		}
		keys := d.Sections[section]
		domains := append([]string{meta.Domain}, meta.AlsoDomains...)
		for _, k := range sortedKeys(keys) {
			for _, domain := range domains {
				out = append(out, DesiredSetting{
					Domain:  domain,
					Key:     k,
					Value:   keys[k],
					Section: section,
					Scope:   meta.Scope,
				})
			}
		}
	}
	domains := sortedKeysMap(d.Custom)
	for _, domain := range domains {
		keys := d.Custom[domain]
		for _, k := range sortedKeys(keys) {
			out = append(out, DesiredSetting{
				Domain:  domain,
				Key:     k,
				Value:   keys[k],
				Section: "custom",
				Scope:   "user",
			})
		}
	}
	return out
}

func sortedKeys(m map[string]SettingValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysMap(m map[string]map[string]SettingValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
