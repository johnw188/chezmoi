package chezmoi

// FIXME empty files
// FIXME data command

import (
	"archive/tar"
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
	Archive(*tar.Writer, *tar.Header, os.FileMode) error
	Mode() os.FileMode
	Equal(EntryState) (bool, error)
	Path() string
	Write(Mutator, os.FileMode, string) error
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
		return mutator.Chmod(currentDirState.path, d.mode&os.ModePerm&^umask)
	}
	if currentState != nil {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return d.Write(mutator, umask, targetPath)
}

// Archive writes d to w.
func (d *DirState) Archive(w *tar.Writer, headerTemplate *tar.Header, umask os.FileMode) error {
	header := *headerTemplate
	header.Typeflag = tar.TypeDir
	header.Name = d.path
	header.Mode = int64(d.mode & os.ModePerm &^ umask)
	return w.WriteHeader(&header)
}

// Equal returns true if d is equal to other. It does not recurse.
func (d *DirState) Equal(other EntryState) (bool, error) {
	otherD, ok := other.(*DirState)
	if !ok {
		return false, nil
	}
	return d.mode == otherD.mode, nil
}

// Mode returns d's mode.
func (d *DirState) Mode() os.FileMode {
	return d.mode
}

// Path returns d's path.
func (d *DirState) Path() string {
	return d.path
}

// Write writes d to fs.
func (d *DirState) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	return mutator.Mkdir(targetPath, d.mode&os.ModePerm&^umask)
}

// NewFileState returns a new FileState populated with path and info on fs.
func NewFileState(fs vfs.FS, path string, info os.FileInfo) *FileState {
	return &FileState{
		path: path,
		mode: info.Mode(),
		contentsFunc: func() ([]byte, error) {
			return fs.ReadFile(path)
		},
	}
}

// Apply updates targetPath to be f using mutator.
func (f *FileState) Apply(mutator Mutator, umask os.FileMode, targetPath string, currentState EntryState) error {
	if currentFileState, ok := currentState.(*FileState); ok {
		contentsSHA256, err := f.ContentsSHA256()
		if err != nil {
			return err
		}
		targetContentsSHA256, err := currentFileState.ContentsSHA256()
		if err != nil {
			return err
		}
		if bytes.Equal(contentsSHA256, targetContentsSHA256) {
			if f.mode&^umask == currentFileState.mode&^umask {
				return nil
			}
			return mutator.Chmod(f.path, f.mode&os.ModePerm&^umask)
		}
	}
	if _, ok := currentState.(*FileState); !ok {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	// FIXME empty
	return f.Write(mutator, umask, targetPath)
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

// Mode returns f's mode.
func (f *FileState) Mode() os.FileMode {
	return f.mode
}

// Path returns f's path.
func (f *FileState) Path() string {
	return f.path
}

// Write writes f to fs.
func (f *FileState) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	contents, err := f.Contents()
	if err != nil {
		return err
	}
	return mutator.WriteFile(targetPath, contents, f.mode&os.ModePerm&^umask, nil)
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

// Mode returns s's mode.
func (s *SymlinkState) Mode() os.FileMode {
	return s.mode
}

// Path returns d's path.
func (s *SymlinkState) Path() string {
	return s.path
}

// Write writes s to fs.
func (s *SymlinkState) Write(mutator Mutator, umask os.FileMode, targetPath string) error {
	linkname, err := s.Linkname()
	if err != nil {
		return err
	}
	return mutator.WriteSymlink(linkname, targetPath)
}
