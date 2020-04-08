package chezmoi

import (
	"os"
	"os/exec"

	"github.com/google/renameio"
	vfs "github.com/twpayne/go-vfs"
)

// An FSDestDir is a DestDir on an vfs.FS.
type FSDestDir struct {
	vfs.FS
	devCache     map[string]uint // devCache maps directories to device numbers.
	tempDirCache map[uint]string // tempDir maps device numbers to renameio temporary directories.
}

// NewFSDestDir returns a DestDir that acts on fs.
func NewFSDestDir(fs vfs.FS) *FSDestDir {
	return &FSDestDir{
		FS:           fs,
		devCache:     make(map[string]uint),
		tempDirCache: make(map[uint]string),
	}
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *FSDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// RunCmd implements DestDir.RunCmd.
func (d *FSDestDir) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *FSDestDir) WriteSymlink(oldname, newname string) error {
	// Special case: if writing to the real filesystem, use github.com/google/renameio
	if d.FS == vfs.OSFS {
		return renameio.Symlink(oldname, newname)
	}
	if err := d.FS.RemoveAll(newname); err != nil && !os.IsNotExist(err) {
		return err
	}
	return d.FS.Symlink(oldname, newname)
}
