package plan

import (
	"sort"
	"strconv"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildMasPlan computes install/remove actions for Mac App Store apps.
// Matching is by numeric App Store ID. Action.Name is the config display name
// for installs and the discovered name for removes; Action.Value is the ID.
//
// When packages.mas is not configured (MasConfigured=false), the plan is empty.
// When managed, undeclared installed apps always produce mas_remove (uninstall_mode
// filtering happens at apply time, matching brew).
//
// Action order: installs then removes; each group sorted by ascending ID.
func BuildMasPlan(desired config.Packages, current discovery.MasState) Plan {
	if !desired.MasConfigured {
		return Plan{}
	}

	desiredByID := make(map[int64]config.MasApp, len(desired.Mas))
	for _, app := range desired.Mas {
		desiredByID[app.ID] = app
	}
	currentByID := make(map[int64]discovery.MasInstalled, len(current.Apps))
	for _, app := range current.Apps {
		currentByID[app.ID] = app
	}

	var installIDs []int64
	for id := range desiredByID {
		if _, ok := currentByID[id]; !ok {
			installIDs = append(installIDs, id)
		}
	}
	sort.Slice(installIDs, func(i, j int) bool { return installIDs[i] < installIDs[j] })

	var removeIDs []int64
	for id := range currentByID {
		if _, ok := desiredByID[id]; !ok {
			removeIDs = append(removeIDs, id)
		}
	}
	sort.Slice(removeIDs, func(i, j int) bool { return removeIDs[i] < removeIDs[j] })

	actions := make([]Action, 0, len(installIDs)+len(removeIDs))
	for _, id := range installIDs {
		app := desiredByID[id]
		actions = append(actions, Action{
			Type:  ActionMasInstall,
			Name:  app.Name,
			Value: strconv.FormatInt(id, 10),
		})
	}
	for _, id := range removeIDs {
		app := currentByID[id]
		actions = append(actions, Action{
			Type:  ActionMasRemove,
			Name:  app.Name,
			Value: strconv.FormatInt(id, 10),
		})
	}
	return Plan{Actions: actions}
}
