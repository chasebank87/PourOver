package discovery

import (
	"context"
	"fmt"

	"github.com/chasebank87/PourOver/internal/config"
)

// SnapshotEntry is one catalog key whose current defaults value was readable.
type SnapshotEntry struct {
	Section string
	Key     string
	Domain  string
	Value   config.SettingValue
}

// SnapshotCatalogDefaults reads current values for catalog DesiredSettings.
// Consecutive entries that share Section+Key (primary domain then AlsoDomains)
// collapse to a single SnapshotEntry using the first successful read.
// Missing keys and parse failures are skipped; parse failures append warnings.
func SnapshotCatalogDefaults(ctx context.Context, runner DefaultsRunner, desired []config.DesiredSetting) ([]SnapshotEntry, []string, error) {
	var out []SnapshotEntry
	var warnings []string

	for i := 0; i < len(desired); {
		group := desired[i]
		j := i + 1
		for j < len(desired) && desired[j].Section == group.Section && desired[j].Key == group.Key {
			j++
		}
		candidates := desired[i:j]
		i = j

		entry, warn, err := snapshotOneKey(ctx, runner, candidates)
		if err != nil {
			return nil, nil, err
		}
		if warn != "" {
			warnings = append(warnings, warn)
		}
		if entry != nil {
			out = append(out, *entry)
		}
	}
	return out, warnings, nil
}

func snapshotOneKey(ctx context.Context, runner DefaultsRunner, candidates []config.DesiredSetting) (*SnapshotEntry, string, error) {
	var parseWarn string
	for _, d := range candidates {
		raw, found, err := readDefault(ctx, runner, d.Domain, d.Key)
		if err != nil {
			return nil, "", err
		}
		if !found {
			continue
		}
		val, err := ParseDefaultsRead(raw, d.Value.Kind)
		if err != nil {
			if parseWarn == "" {
				parseWarn = fmt.Sprintf("%s.%s (%s): %v", d.Section, d.Key, d.Domain, err)
			}
			continue
		}
		return &SnapshotEntry{
			Section: d.Section,
			Key:     d.Key,
			Domain:  d.Domain,
			Value:   val,
		}, "", nil
	}
	return nil, parseWarn, nil
}
