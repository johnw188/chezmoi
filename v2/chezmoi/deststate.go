package chezmoi

// FIXME data command

import (
	"bytes"
	"os"

	vfs "github.com/twpayne/go-vfs"
)

// An DestStateEntry represents the state of an entry in the destination state.
type DestStateEntry interface {
	Apply(Mutator, os.FileMode, string, DestStateEntry) error
	Equal(DestStateEntry) (bool, error)
}

// A DestStateDir represents the state of a directory in the destination state.
type DestStateDir struct {
	path string
	mode os.FileMode
}

// A DestStateFile represents the state of a file in the destination state.
type DestStateFile struct {
	path           string
	mode           os.FileMode
	empty          bool
	contentsFunc   func() ([]byte, error)
	contents       []byte
	contentsSHA256 []byte
	contentsErr    error
}

// A DestDirSymlink represents the state of a symlink in the destination state.
type DestDirSymlink struct {
	path         string
	mode         os.FileMode
	linknameFunc func() (string, error)
	linkname     string
	linknameErr  error
}

var emptySHA256 = sha256Sum(nil)

// NewEntryState returns a new EntryState populated with path from fs.
func NewEntryState(fs vfs.FS, path string) (DestStateEntry, error) {
	info, err := fs.Lstat(path)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return NewDestStateFromInfo(fs, path, info)
}

// NewDestStateFromInfo returns a new EntryState populated with path and info on fs.
func NewDestStateFromInfo(fs vfs.FS, path string, info os.FileInfo) (DestStateEntry, error) {
	switch info.Mode() & os.ModeType {
	case 0:
		empty := info.Size() == 0
		return newDestStateFileFromInfo(fs, path, info, empty), nil
	case os.ModeDir:
		return newDestStateDirFromInfo(fs, path, info), nil
	case os.ModeSymlink:
		return newDestStateSymlinkFromInfo(fs, path, info), nil
	default:
		return nil, &unsupportedFileTypeError{
			path: path,
			mode: info.Mode(),
		}
	}
}

// newDestStateDirFromInfo returns a new DirState populated with path and info on fs.
func newDestStateDirFromInfo(fs vfs.FS, path string, info os.FileInfo) *DestStateDir {
	return &DestStateDir{
		path: path,
		mode: info.Mode(),
	}
}

// Apply updates targetPath to be d using mutator.
func (d *DestStateDir) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState DestStateEntry) error {
	if currentDirState, ok := currentState.(*DestStateDir); ok {
		if currentDirState.mode&os.ModePerm&^umask == d.mode&os.ModePerm&^umask {
			return nil
		}
		return mutator.Chmod(targetPath, d.mode&os.ModePerm&^umask)
	}
	if currentState != nil {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return d.Write(mutator, umask, targetPath)
}

// Equal returns true if d is equal to other. It does not recurse.
func (d *DestStateDir) Equal(other DestStateEntry) (bool, error) {
	otherD, ok := other.(*DestStateDir)
	if !ok {
		return false, nil
	}
	return d.mode == otherD.mode, nil
}

// Write writes d to fs.
func (d *DestStateDir) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	return mutator.Mkdir(targetPath, d.mode&os.ModePerm&^umask)
}

// newDestStateFileFromInfo returns a new FileState populated with path and info on fs.
func newDestStateFileFromInfo(fs vfs.FS, path string, info os.FileInfo, empty bool) *DestStateFile {
	return &DestStateFile{
		path:  path,
		mode:  info.Mode(),
		empty: empty,
		contentsFunc: func() ([]byte, error) {
			return fs.ReadFile(path)
		},
	}
}

// Apply updates targetPath to be f using mutator.
func (f *DestStateFile) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState DestStateEntry) error {
	// FIXME tidy up the logic here. The fundamental problem is that
	// mutator.WriteFile only sets the specified permissions when writing a new
	// file. The solution is probably to update mutator.WriteFile to remove the
	// file first if the permissions don't match.
	var targetContents []byte
	if currentFileState, ok := currentState.(*DestStateFile); ok {
		contentsSHA256, err := f.ContentsSHA256()
		if err != nil {
			return err
		}
		targetContentsSHA256, err := currentFileState.ContentsSHA256()
		if err != nil {
			return err
		}
		if bytes.Equal(contentsSHA256, emptySHA256) && !f.empty {
			return mutator.RemoveAll(targetPath)
		}
		if f.mode&os.ModePerm&^umask != currentFileState.mode&os.ModePerm&^umask {
			if err := mutator.Chmod(targetPath, f.mode&os.ModePerm&^umask); err != nil {
				return err
			}
		}
		if bytes.Equal(contentsSHA256, targetContentsSHA256) {
			return nil
		}
		targetContents, err = currentFileState.Contents()
		if err != nil {
			return err
		}
	} else if currentState == nil && !f.empty {
		contentsSHA256, err := f.ContentsSHA256()
		if err != nil {
			return err
		}
		if bytes.Equal(contentsSHA256, emptySHA256) {
			return nil
		}
	}
	if _, ok := currentState.(*DestStateFile); !ok {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return f.Write(mutator, umask, targetPath, targetContents)
}

// Contents returns e's contents.
func (f *DestStateFile) Contents() ([]byte, error) {
	if f.contentsFunc != nil {
		f.contents, f.contentsErr = f.contentsFunc()
		f.contentsFunc = nil
		if f.contentsErr == nil {
			f.contentsSHA256 = sha256Sum(f.contents)
		}
	}
	return f.contents, f.contentsErr
}

// ContentsSHA256 returns the SHA256 sum of f's contents.
func (f *DestStateFile) ContentsSHA256() ([]byte, error) {
	if f.contentsSHA256 == nil {
		if _, err := f.Contents(); err != nil {
			return nil, err
		}
		f.contentsSHA256 = sha256Sum(f.contents)
	}
	return f.contentsSHA256, nil
}

// Equal returns true if f equals other.
func (f *DestStateFile) Equal(other DestStateEntry) (bool, error) {
	contentsSHA256, err := f.ContentsSHA256()
	if err != nil {
		return false, err
	}
	if other == nil && bytes.Equal(contentsSHA256, emptySHA256) && !f.empty {
		return true, nil
	}
	otherF, ok := other.(*DestStateFile)
	if !ok {
		return false, nil
	}
	if f.mode != otherF.mode {
		return false, nil
	}
	otherContentsSHA256, err := otherF.ContentsSHA256()
	if err != nil {
		return false, err
	}
	return bytes.Equal(contentsSHA256, otherContentsSHA256), nil
}

// Write writes f to fs.
func (f *DestStateFile) Write(mutator Mutator, umask os.FileMode, targetPath string, currentContents []byte) error {
	contents, err := f.Contents()
	if err != nil {
		return err
	}
	if len(contents) == 0 && !f.empty {
		return nil
	}
	return mutator.WriteFile(targetPath, contents, f.mode&os.ModePerm&^umask, currentContents)
}

// newDestStateSymlinkFromInfo returns a new SymlinkState populated with path and info on
// fs.
func newDestStateSymlinkFromInfo(fs vfs.FS, path string, info os.FileInfo) *DestDirSymlink {
	return &DestDirSymlink{
		path: path,
		mode: info.Mode(),
		linknameFunc: func() (string, error) {
			return fs.Readlink(path)
		},
	}
}

// Apply updates target to be s using mutator.
func (s *DestDirSymlink) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState DestStateEntry) error {
	if targetS, ok := currentState.(*DestDirSymlink); ok {
		linkname, err := s.Linkname()
		if err != nil {
			return err
		}
		targetLinkname, err := targetS.Linkname()
		if err != nil {
			return err
		}
		if linkname == targetLinkname {
			return nil
		}
	}
	if _, ok := currentState.(*DestDirSymlink); !ok {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return s.Write(mutator, umask, targetPath)
}

// Equal returns true if s is equal to other.
func (s *DestDirSymlink) Equal(other DestStateEntry) (bool, error) {
	otherS, ok := other.(*DestDirSymlink)
	if !ok {
		return false, nil
	}
	linkname, err := s.Linkname()
	if err != nil {
		return false, err
	}
	otherLinkname, err := otherS.Linkname()
	if err != nil {
		return false, err
	}
	return linkname == otherLinkname, nil
}

// Linkname returns s's linkname.
func (s *DestDirSymlink) Linkname() (string, error) {
	if s.linknameFunc != nil {
		s.linkname, s.linknameErr = s.linknameFunc()
	}
	return s.linkname, s.linknameErr
}

// Write writes s to fs.
func (s *DestDirSymlink) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	linkname, err := s.Linkname()
	if err != nil {
		return err
	}
	return mutator.WriteSymlink(linkname, targetPath)
}
