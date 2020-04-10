package chezmoi

import (
	"os"
)

// A SourceStateEntry represents the state of an entry in the source state.
type SourceStateEntry interface {
	Evaluate() error
	Path() string
	TargetStateEntry() TargetStateEntry
	Write(fs FileSystem, umask os.FileMode) error
}

// A SourceStateDir represents the state of a directory in the source state.
type SourceStateDir struct {
	path             string
	attributes       DirAttributes
	targetStateEntry TargetStateEntry
}

// A SourceStateFile represents the state of a file in the source state.
type SourceStateFile struct {
	*lazyContents
	path             string
	attributes       FileAttributes
	targetStateEntry TargetStateEntry
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateDir) Evaluate() error {
	return nil
}

// Path returns s's path.
func (s *SourceStateDir) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateDir) TargetStateEntry() TargetStateEntry {
	return s.targetStateEntry
}

// Write writes s to sourceStateDir.
func (s *SourceStateDir) Write(sourceStateDir FileSystem, umask os.FileMode) error {
	return sourceStateDir.Mkdir(s.path, 0o777&^umask)
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateFile) Evaluate() error {
	_, err := s.ContentsSHA256()
	return err
}

// Path returns s's path.
func (s *SourceStateFile) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateFile) TargetStateEntry() TargetStateEntry {
	return s.targetStateEntry
}

// Write writes s to sourceStateDir.
func (s *SourceStateFile) Write(sourceStateDir FileSystem, umask os.FileMode) error {
	contents, err := s.Contents()
	if err != nil {
		return err
	}
	currContents, err := sourceStateDir.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return sourceStateDir.WriteFile(s.path, contents, 0o666&^umask, currContents)
}
