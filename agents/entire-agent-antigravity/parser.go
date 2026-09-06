package main

// SessionParser converts a specific Antigravity input format into SessionIR.
type SessionParser interface {
	Parse(root string) (*SessionIR, error)
}
