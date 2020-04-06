package chezmoi

// FIXME data command

import (
	"bytes"
	"crypto/sha256"
	"os"

	vfs "github.com/twpayne/go-vfs"
)

// A StatFunc is a function like os.Stat or os.Lstat.
type StatFunc func(string) (os.FileInfo, error)

// An EntryState represents the state of an entry.
type EntryState interface {
	Apply(Mutator, os.FileMode, string, EntryState) error
	Equal(EntryState) (bool, error)
}

// A DirState represents the state of a directory.
type DirState struct {
	path string
	mode os.FileMode
}

// A FileState represents the state of a file.
type FileState struct {
	path           string
	mode           os.FileMode
	empty          bool
	contentsFunc   func() ([]byte, error)
	contents       []byte
	contentsSHA256 []byte
	contentsErr    error
}

// A SymlinkState represents the state of a symlink.
type SymlinkState struct {
	path         string
	mode         os.FileMode
	linknameFunc func() (string, error)
	linkname     string
	linknameErr  error
}

var emptySHA256 = sha256Sum(nil)

// NewEntryState returns a new EntryState populated with path from fs.
func NewEntryState(fs vfs.FS, statFunc StatFunc, path string) (EntryState, error) {
	info, err := statFunc(path)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return NewEntryStateWithInfo(fs, path, info)
}

// NewEntryStateWithInfo returns a new EntryState populated with path and info on fs.
func NewEntryStateWithInfo(fs vfs.FS, path string, info os.FileInfo) (EntryState, error) {
	switch info.Mode() & os.ModeType {
	case 0:
		empty := info.Size() == 0
		return NewFileState(fs, path, info, empty), nil
	case os.ModeDir:
		return NewDirState(fs, path, info), nil
	case os.ModeSymlink:
		return NewSymlinkState(fs, path, info), nil
	default:
		return nil, &unsupportedFileTypeError{
			path: path,
			mode: info.Mode(),
		}
	}
}

// NewDirState returns a new DirState populated with path and info on fs.
func NewDirState(fs vfs.FS, path string, info os.FileInfo) *DirState {
	return &DirState{
		path: path,
		mode: info.Mode(),
	}
}

// Apply updates targetPath to be d using mutator.
func (d *DirState) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState EntryState) error {
	if currentDirState, ok := currentState.(*DirState); ok {
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
func (d *DirState) Equal(other EntryState) (bool, error) {
	otherD, ok := other.(*DirState)
	if !ok {
		return false, nil
	}
	return d.mode == otherD.mode, nil
}

// Write writes d to fs.
func (d *DirState) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	return mutator.Mkdir(targetPath, d.mode&os.ModePerm&^umask)
}

// NewFileState returns a new FileState populated with path and info on fs.
func NewFileState(fs vfs.FS, path string, info os.FileInfo, empty bool) *FileState {
	return &FileState{
		path:  path,
		mode:  info.Mode(),
		empty: empty,
		contentsFunc: func() ([]byte, error) {
			return fs.ReadFile(path)
		},
	}
}

// Apply updates targetPath to be f using mutator.
func (f *FileState) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState EntryState) error {
	// FIXME tidy up the logic here. The fundamental problem is that
	// mutator.WriteFile only sets the specified permissions when writing a new
	// file. The solution is probably to update mutator.WriteFile to remove the
	// file first if the permissions don't match.
	var targetContents []byte
	if currentFileState, ok := currentState.(*FileState); ok {
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
	if _, ok := currentState.(*FileState); !ok {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return f.Write(mutator, umask, targetPath, targetContents)
}

// Contents returns e's contents.
func (f *FileState) Contents() ([]byte, error) {
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
func (f *FileState) ContentsSHA256() ([]byte, error) {
	if f.contentsSHA256 == nil {
		if _, err := f.Contents(); err != nil {
			return nil, err
		}
		f.contentsSHA256 = sha256Sum(f.contents)
	}
	return f.contentsSHA256, nil
}

// Equal returns true if f equals other.
func (f *FileState) Equal(other EntryState) (bool, error) {
	contentsSHA256, err := f.ContentsSHA256()
	if err != nil {
		return false, err
	}
	if other == nil && bytes.Equal(contentsSHA256, emptySHA256) && !f.empty {
		return true, nil
	}
	otherF, ok := other.(*FileState)
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
func (f *FileState) Write(mutator Mutator, umask os.FileMode, targetPath string, currentContents []byte) error {
	contents, err := f.Contents()
	if err != nil {
		return err
	}
	if len(contents) == 0 && !f.empty {
		return nil
	}
	return mutator.WriteFile(targetPath, contents, f.mode&os.ModePerm&^umask, currentContents)
}

// NewSymlinkState returns a new SymlinkState populated with path and info on
// fs.
func NewSymlinkState(fs vfs.FS, path string, info os.FileInfo) *SymlinkState {
	return &SymlinkState{
		path: path,
		mode: info.Mode(),
		linknameFunc: func() (string, error) {
			return fs.Readlink(path)
		},
	}
}

// Apply updates target to be s using mutator.
func (s *SymlinkState) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState EntryState) error {
	if targetS, ok := currentState.(*SymlinkState); ok {
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
	if _, ok := currentState.(*SymlinkState); !ok {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return s.Write(mutator, umask, targetPath)
}

// Equal returns true if s is equal to other.
func (s *SymlinkState) Equal(other EntryState) (bool, error) {
	otherS, ok := other.(*SymlinkState)
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
func (s *SymlinkState) Linkname() (string, error) {
	if s.linknameFunc != nil {
		s.linkname, s.linknameErr = s.linknameFunc()
	}
	return s.linkname, s.linknameErr
}

// Write writes s to fs.
func (s *SymlinkState) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	linkname, err := s.Linkname()
	if err != nil {
		return err
	}
	return mutator.WriteSymlink(linkname, targetPath)
}

func sha256Sum(data []byte) []byte {
	sha256SumArr := sha256.Sum256(data)
	return sha256SumArr[:]
}
