package chezmoi

import (
	"os"
)

// A SourceStateEntry represents the state of an entry in the source state.
type SourceStateEntry interface {
	Path() string
	TargetStateEntry(sourceStateDir DestDir, umask os.FileMode) TargetStateEntry
	Write(sourceStateDir DestDir, umask os.FileMode) error
}

// A SourceStateDir represents the state of a directory in the source state.
type SourceStateDir struct {
	path       string
	attributes DirAttributes
}

// A SourceStateFile represents the state of a file in the source state.
type SourceStateFile struct {
	path       string
	attributes FileAttributes
	*lazyContents
}

// NewSourceStateDir returns a new SourceStateDir.
func NewSourceStateDir(path string, attributes DirAttributes) *SourceStateDir {
	return &SourceStateDir{
		path:       path,
		attributes: attributes,
	}
}

// Path returns s's path.
func (s *SourceStateDir) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateDir) TargetStateEntry(sourceStateDir DestDir, umask os.FileMode) TargetStateEntry {
	perm := os.FileMode(0o777)
	if s.attributes.Private {
		perm &^= 0o77
	}
	return &TargetStateDir{
		perm:  perm &^ umask,
		exact: s.attributes.Exact,
	}
}

// Write writes s to sourceStateDir.
func (s *SourceStateDir) Write(sourceStateDir DestDir, umask os.FileMode) error {
	return sourceStateDir.Mkdir(s.path, 0o777&^umask)
}

// NewSourceStateFile returns a new SourceStateFile.
func NewSourceStateFile(path string, attributes FileAttributes, contents []byte) *SourceStateFile {
	return &SourceStateFile{
		path:       path,
		attributes: attributes,
		lazyContents: &lazyContents{
			contents: contents,
		},
	}
}

// Path returns s's path.
func (s *SourceStateFile) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateFile) TargetStateEntry(sourceStateDir DirReader, umask os.FileMode) TargetStateEntry {
	switch s.attributes.Type {
	case SourceFileTypeFile:
		perm := os.FileMode(0o666)
		if s.attributes.Executable {
			perm |= 0o111
		}
		if s.attributes.Private {
			perm &^= 0o77
		}
		return &TargetStateFile{
			perm: perm &^ umask,
			lazyContents: &lazyContents{
				contentsFunc: func() ([]byte, error) {
					return sourceStateDir.ReadFile(s.path)
				},
			},
		}
	case SourceFileTypeScript:
		return &TargetStateScript{
			name: s.attributes.Name,
			lazyContents: &lazyContents{
				contentsFunc: func() ([]byte, error) {
					return sourceStateDir.ReadFile(s.path)
				},
			},
		}
	case SourceFileTypeSymlink:
		return &TargetStateSymlink{
			lazyLinkname: &lazyLinkname{
				linknameFunc: func() (string, error) {
					linknameBytes, err := sourceStateDir.ReadFile(s.path)
					if err != nil {
						return "", err
					}
					return string(linknameBytes), nil
				},
			},
		}
	default:
		return nil
	}
}

// Write writes s to sourceStateDir.
func (s *SourceStateFile) Write(sourceStateDir DestDir, umask os.FileMode) error {
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
