package engine

// Confirmer asks the user to confirm a destructive or significant action.
type Confirmer interface {
	Confirm(prompt string) bool
}
