package plan

// ActionType describes a single reconciliation step.
type ActionType string

const (
	ActionTapAdd         ActionType = "tap_add"
	ActionTapTrust       ActionType = "tap_trust"
	ActionTapRemove      ActionType = "tap_remove"
	ActionFormulaInstall ActionType = "formula_install"
	ActionFormulaRemove  ActionType = "formula_remove"
	ActionFormulaUpgrade ActionType = "formula_upgrade"
	ActionCaskInstall    ActionType = "cask_install"
	ActionCaskRemove     ActionType = "cask_remove"
	ActionCaskUpgrade    ActionType = "cask_upgrade"
	ActionCaskRename     ActionType = "cask_rename" // advise: Name is old token, Value is new token
	ActionLinkCreate     ActionType = "link_create"
	ActionLinkUpdate     ActionType = "link_update"
	ActionLinkReplace    ActionType = "link_replace" // backup existing target, then create link
	ActionManagedCopy    ActionType = "managed_copy"
	ActionFileUnlink     ActionType = "file_unlink"
	ActionDefaultsWrite  ActionType = "defaults_write"
)

// Action is one change to apply.
type Action struct {
	Type   ActionType `json:"type"`
	Name   string     `json:"name"`
	Source string     `json:"source,omitempty"`
	Domain string     `json:"domain,omitempty"`
	Key    string     `json:"key,omitempty"`
	Value  string     `json:"value,omitempty"`
	Kind   string     `json:"kind,omitempty"`
}

// Plan is an ordered list of actions to reconcile state.
type Plan struct {
	Actions []Action `json:"actions"`
}
