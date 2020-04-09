package chezmoi

import (
	"os"
)

// A SourceStateEntry represents the state of an entry in the source state.
type SourceStateEntry interface {
	Path() string
	TargetStateEntry(umask os.FileMode) TargetStateEntry
}

// A SourceStateDir represents the state of a directory in the source state.
type SourceStateDir struct {
	path       string
	attributes DirAttributes
}

// A SourceStateFile represents the state of a file in the source state.
type SourceStateFile struct {
	dirReader  DirReader
	path       string
	attributes FileAttributes
	*lazyContents
}

// Path returns s's path.
func (s *SourceStateDir) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateDir) TargetStateEntry(umask os.FileMode) TargetStateEntry {
	perm := os.FileMode(0o777)
	if s.attributes.Private {
		perm &^= 0o77
	}
	return &TargetStateDir{
		perm:  perm &^ umask,
		exact: s.attributes.Exact,
	}
}

// Path returns s's path.
func (s *SourceStateFile) Path() string {
	return s.path
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateFile) TargetStateEntry(umask os.FileMode) TargetStateEntry {
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
					return s.dirReader.ReadFile(s.path)
				},
			},
		}
	case SourceFileTypeScript:
		return &TargetStateScript{
			name: s.attributes.Name,
			lazyContents: &lazyContents{
				contentsFunc: func() ([]byte, error) {
					return s.dirReader.ReadFile(s.path)
				},
			},
		}
	case SourceFileTypeSymlink:
		return &TargetStateSymlink{
			lazyLinkname: &lazyLinkname{
				linknameFunc: func() (string, error) {
					linknameBytes, err := s.dirReader.ReadFile(s.path)
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
