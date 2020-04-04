package chezmoi

// FIXME empty files
// FIXME data command
// FIXME should be able to remove some calls to RemoveAll

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"

	vfs "github.com/twpayne/go-vfs"
)

// A StatFunc is a function like os.Stat or os.Lstat.
type StatFunc func(string) (os.FileInfo, error)

// An entryState contains fields common across all EntryStates.
type entryState struct {
	path string
	mode os.FileMode
}

// An EntryState represents the state of an entry.
type EntryState interface {
	Apply(Mutator, os.FileMode, EntryState) error
	Archive(*tar.Writer, *tar.Header, os.FileMode) error
	Mode() os.FileMode
	Equal(EntryState) (bool, error)
	Path() string
	Write(Mutator, os.FileMode) error
}

// A DirState represents the state of a directory.
type DirState struct {
	entryState
	entriesFunc func() ([]os.FileInfo, error)
	entries     []os.FileInfo
	entriesErr  error
}

// A FileState represents the state of a file.
type FileState struct {
	entryState
	contentsFunc   func() ([]byte, error)
	contents       []byte
	contentsSHA256 []byte
	contentsErr    error
}

// A SymlinkState represents the state of a symlink.
type SymlinkState struct {
	entryState
	linknameFunc func() (string, error)
	linkname     string
	linknameErr  error
}

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
		return NewFileState(fs, path, info), nil
	case os.ModeDir:
		return NewDirState(fs, path, info), nil
	case os.ModeSymlink:
		return NewSymlinkState(fs, path, info), nil
	default:
		return nil, fmt.Errorf("%s: unsupported file type %d", path, info.Mode()&os.ModeType)
	}
}

// Path return e's path.
func (e *entryState) Path() string {
	return e.path
}

// Mode returns e's mode.
func (e *entryState) Mode() os.FileMode {
	return e.mode
}

// NewDirState returns a new DirState populated with path and info on fs.
func NewDirState(fs vfs.FS, path string, info os.FileInfo) *DirState {
	return &DirState{
		entryState: entryState{
			path: path,
			mode: info.Mode(),
		},
		entriesFunc: func() ([]os.FileInfo, error) {
			return fs.ReadDir(path)
		},
	}
}

// Apply updates target to be d using mutator.
func (d *DirState) Apply(mutator Mutator, umask os.FileMode, target EntryState) error {
	if targetD, ok := target.(*DirState); ok {
		if targetD.mode&os.ModePerm&^umask == d.mode&os.ModePerm&^umask {
			return nil
		}
		return mutator.Chmod(targetD.path, d.mode&os.ModePerm&^umask)
	}
	if err := mutator.RemoveAll(d.path); err != nil {
		return err
	}
	return d.Write(mutator, umask)
}

// Archive writes d to w.
func (d *DirState) Archive(w *tar.Writer, headerTemplate *tar.Header, umask os.FileMode) error {
	header := *headerTemplate
	header.Typeflag = tar.TypeDir
	header.Name = d.path
	header.Mode = int64(d.mode & os.ModePerm &^ umask)
	return w.WriteHeader(&header)
}

// Entries returns d's entries.
func (d *DirState) Entries() ([]os.FileInfo, error) {
	if d.entriesFunc != nil {
		d.entries, d.entriesErr = d.entriesFunc()
		d.entriesFunc = nil
	}
	return d.entries, d.entriesErr
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
func (d *DirState) Write(mutator Mutator, umask os.FileMode) error {
	return mutator.Mkdir(d.path, d.mode&os.ModePerm&^umask)
}

// NewFileState returns a new FileState populated with path and info on fs.
func NewFileState(fs vfs.FS, path string, info os.FileInfo) *FileState {
	return &FileState{
		entryState: entryState{
			path: path,
			mode: info.Mode(),
		},
		contentsFunc: func() ([]byte, error) {
			return fs.ReadFile(path)
		},
	}
}

// Apply updates target to be f using mutator.
func (f *FileState) Apply(mutator Mutator, umask os.FileMode, target EntryState) error {
	if targetF, ok := target.(*FileState); ok {
		contentsSHA256, err := f.ContentsSHA256()
		if err != nil {
			return err
		}
		targetContentsSHA256, err := targetF.ContentsSHA256()
		if err != nil {
			return err
		}
		if bytes.Equal(contentsSHA256, targetContentsSHA256) {
			if f.mode&^umask == targetF.mode&^umask {
				return nil
			}
			return mutator.Chmod(f.path, f.mode&os.ModePerm&^umask)
		}
	}
	if err := mutator.RemoveAll(f.path); err != nil {
		return err
	}
	return f.Write(mutator, umask) // FIXME
}

// Archive writes f to w.
func (f *FileState) Archive(w *tar.Writer, headerTemplate *tar.Header, umask os.FileMode) error {
	contents, err := f.Contents()
	if err != nil {
		return err
	}
	header := *headerTemplate
	header.Typeflag = tar.TypeReg
	header.Name = f.path
	header.Size = int64(len(contents))
	header.Mode = int64(f.mode & os.ModePerm &^ umask)
	if err := w.WriteHeader(&header); err != nil {
		return err
	}
	_, err = w.Write(contents)
	return err
}

// Contents returns e's contents.
func (f *FileState) Contents() ([]byte, error) {
	if f.contentsFunc != nil {
		f.contents, f.contentsErr = f.contentsFunc()
		f.contentsFunc = nil
		if f.contentsErr == nil {
			contentsSHA256 := sha256.Sum256(f.contents)
			f.contentsSHA256 = contentsSHA256[:]
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
		contentsSHA256 := sha256.Sum256(f.contents)
		f.contentsSHA256 = contentsSHA256[:]
	}
	return f.contentsSHA256, nil
}

// Equal returns true if f equals other.
func (f *FileState) Equal(other EntryState) (bool, error) {
	otherF, ok := other.(*FileState)
	if !ok {
		return false, nil
	}
	if f.mode != otherF.mode {
		return false, nil
	}
	contentsSHA256, err := f.ContentsSHA256()
	if err != nil {
		return false, err
	}
	otherContentsSHA256, err := otherF.ContentsSHA256()
	if err != nil {
		return false, err
	}
	return bytes.Equal(contentsSHA256, otherContentsSHA256), nil
}

// Write writes f to fs.
func (f *FileState) Write(mutator Mutator, umask os.FileMode) error {
	contents, err := f.Contents()
	if err != nil {
		return err
	}
	return mutator.WriteFile(f.path, contents, f.mode&os.ModePerm&^umask, nil)
}

// NewSymlinkState returns a new SymlinkState populated with path and info on
// fs.
func NewSymlinkState(fs vfs.FS, path string, info os.FileInfo) *SymlinkState {
	return &SymlinkState{
		entryState: entryState{
			path: path,
			mode: info.Mode(),
		},
		linknameFunc: func() (string, error) {
			return fs.Readlink(path)
		},
	}
}

// Apply updates target to be s using mutator.
func (s *SymlinkState) Apply(mutator Mutator, umask os.FileMode, target EntryState) error {
	if targetS, ok := target.(*SymlinkState); ok {
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
	if err := mutator.RemoveAll(s.path); err != nil {
		return err
	}
	return s.Write(mutator, umask)
}

// Archive writes s to w.
func (s *SymlinkState) Archive(w *tar.Writer, headerTemplate *tar.Header, umask os.FileMode) error {
	linkname, err := s.Linkname()
	if err != nil {
		return err
	}
	header := *headerTemplate
	header.Typeflag = tar.TypeSymlink
	header.Name = s.path
	header.Linkname = linkname
	return w.WriteHeader(&header)
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
func (s *SymlinkState) Write(mutator Mutator, umask os.FileMode) error {
	linkname, err := s.Linkname()
	if err != nil {
		return err
	}
	return mutator.WriteSymlink(linkname, s.path)
}
