package configimport

import (
	"sort"

	"github.com/chasebank87/PourOver/internal/config"
)

// MergePackageLists returns the sorted unique union of existing and discovered
// package names, plus the names that were newly added from discovered.
func MergePackageLists(existing, discovered []string) (merged, added []string) {
	seen := make(map[string]struct{}, len(existing)+len(discovered))
	for _, name := range existing {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		merged = append(merged, name)
	}
	for _, name := range discovered {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		merged = append(merged, name)
		added = append(added, name)
	}
	sort.Strings(merged)
	sort.Strings(added)
	return merged, added
}

// MergeFileLinks keeps existing links in order, then appends imported links
// whose Target is not already declared (new targets sorted by Target).
func MergeFileLinks(existing, imported []config.FileLink) (merged, added []config.FileLink) {
	seen := make(map[string]struct{}, len(existing)+len(imported))
	for _, link := range existing {
		if link.Target == "" {
			continue
		}
		if _, ok := seen[link.Target]; ok {
			continue
		}
		seen[link.Target] = struct{}{}
		merged = append(merged, link)
	}
	var newOnes []config.FileLink
	for _, link := range imported {
		if link.Target == "" {
			continue
		}
		if _, ok := seen[link.Target]; ok {
			continue
		}
		seen[link.Target] = struct{}{}
		newOnes = append(newOnes, link)
	}
	sort.Slice(newOnes, func(i, j int) bool {
		return newOnes[i].Target < newOnes[j].Target
	})
	added = newOnes
	merged = append(merged, newOnes...)
	return merged, added
}

// LinkTargets returns the set of link targets for quick membership checks.
func LinkTargets(links []config.FileLink) map[string]struct{} {
	out := make(map[string]struct{}, len(links))
	for _, link := range links {
		if link.Target != "" {
			out[link.Target] = struct{}{}
		}
	}
	return out
}
