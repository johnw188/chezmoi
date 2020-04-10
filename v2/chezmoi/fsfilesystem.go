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
func (d *FSFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// RunCmd implements FileSystem.RunCmd.
func (d *FSFileSystem) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (d *FSFileSystem) WriteSymlink(oldname, newname string) error {
	// Special case: if writing to the real filesystem, use
	// github.com/google/renameio.
	if d.FS == vfs.OSFS {
		return renameio.Symlink(oldname, newname)
	}
	if err := d.FS.RemoveAll(newname); err != nil && !os.IsNotExist(err) {
		return err
	}
	return d.FS.Symlink(oldname, newname)
}

// WriteFile implements FileSystem.WriteFile.
func (d *FSFileSystem) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	// Special case: if writing to the real filesystem on a non-Windows system,
	// use github.com/google/renameio.
	if d.FS == vfs.OSFS && runtime.GOOS != "windows" {
		dir := path.Dir(name)
		dev, ok := d.devCache[dir]
		if !ok {
			info, err := d.Stat(dir)
			if err != nil {
				return err
			}
			statT, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return errors.New("os.FileInfo.Sys() cannot be converted to a *syscall.Stat_t")
			}
			dev = uint(statT.Dev)
			d.devCache[dir] = dev
		}
		tempDir, ok := d.tempDirCache[dev]
		if !ok {
			tempDir = renameio.TempDir(dir)
			d.tempDirCache[dev] = tempDir
		}
		t, err := renameio.TempFile(tempDir, name)
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
	return d.FS.WriteFile(name, data, perm)
}
