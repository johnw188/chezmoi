package chezmoi

import "os"

// A TargetStateEntry represents the state of an entry in the target state.
type TargetStateEntry interface {
	Equal(DestStateEntry) bool
	Replace(Mutator, os.FileMode, DestStateEntry) error
}

// A TargetStateDir represents the state of a directory in the target state.
type TargetStateDir struct {
	// FIXME
}

// A TargetStateFile represents the state of a file in the target state.
type TargetStateFile struct {
	// FIXME
}

// A TargetStateSymlink represents the state of a symlink in the target state.
type TargetStateSymlink struct {
	// FIXME
}
