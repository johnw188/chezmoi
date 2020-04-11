package chezmoi

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"runtime"
	"syscall"

	"github.com/google/renameio"
	vfs "github.com/twpayne/go-vfs"
)

// An FSFileSystem is a FileSystem on an vfs.FS.
type FSFileSystem struct {
	vfs.FS
	devCache     map[string]uint // devCache maps directories to device numbers.
	tempDirCache map[uint]string // tempDir maps device numbers to renameio temporary directories.
}

// NewFSFileSystem returns a FileSystem that acts on fs.
func NewFSFileSystem(fs vfs.FS) *FSFileSystem {
	return &FSFileSystem{
		FS:           fs,
		devCache:     make(map[string]uint),
		tempDirCache: make(map[uint]string),
	}
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *FSFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// RunCmd implements FileSystem.RunCmd.
func (fs *FSFileSystem) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *FSFileSystem) WriteSymlink(oldname, newname string) error {
	// Special case: if writing to the real filesystem, use
	// github.com/google/renameio.
	if fs.FS == vfs.OSFS {
		return renameio.Symlink(oldname, newname)
	}
	if err := fs.FS.RemoveAll(newname); err != nil && !os.IsNotExist(err) {
		return err
	}
	return fs.FS.Symlink(oldname, newname)
}

// WriteFile implements FileSystem.WriteFile.
func (fs *FSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error {
	// Special case: if writing to the real filesystem on a non-Windows system,
	// use github.com/google/renameio.
	if fs.FS == vfs.OSFS && runtime.GOOS != "windows" {
		dir := path.Dir(filename)
		dev, ok := fs.devCache[dir]
		if !ok {
			info, err := fs.Stat(dir)
			if err != nil {
				return err
			}
			statT, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return errors.New("os.FileInfo.Sys() cannot be converted to a *syscall.Stat_t")
			}
			dev = uint(statT.Dev)
			fs.devCache[dir] = dev
		}
		tempDir, ok := fs.tempDirCache[dev]
		if !ok {
			tempDir = renameio.TempDir(dir)
			fs.tempDirCache[dev] = tempDir
		}
		t, err := renameio.TempFile(tempDir, filename)
		if err != nil {
			return err
		}
		defer func() {
			_ = t.Cleanup()
		}()
		if err := t.Chmod(perm); err != nil {
			return err
		}
		if _, err := t.Write(data); err != nil {
			return err
		}
		return t.CloseAtomicallyReplace()
	}

	// ioutil.WriteFile only sets the permissions when creating a new file. We
	// need to ensure permissions, so we use our own implementation.

	// Create a new file, or truncate any existing one.
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	// From now on, we continue to the end of the function to ensure that
	// f.Close() gets called so we don't leak any file descriptors.

	// Set permissions after truncation but before writing any data, in case the
	// file contained private data before, but before writing the new contents,
	// in case the contents contain private data after.
	err = f.Chmod(perm)

	// If everything is OK so far, write the data.
	if err == nil {
		_, err = f.Write(data)
	}

	// Always call f.Close(), and overwrite the error if so far there is none.
	if err1 := f.Close(); err == nil {
		err = err1
	}

	// Return the first error encounted.
	return err
}
