package chezmoi

import (
	"os"
	"os/exec"
)

// DryRunDestDir is an DestDir that reads from, but does not write to, to a
// wrapped DestDir.
type DryRunDestDir struct {
	d FileSystem
}

// NewDryRunDestDir returns a new DryRunDestDir that wraps m.
func NewDryRunDestDir(d FileSystem) *DryRunDestDir {
	return &DryRunDestDir{
		d: d,
	}
}

// Chmod implements DestDir.Chmod.
func (d *DryRunDestDir) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Glob implements DestDir.Glob.
func (d *DryRunDestDir) Glob(pattern string) ([]string, error) {
	return d.d.Glob(pattern)
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *DryRunDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return d.d.IdempotentCmdOutput(cmd)
}

// Lstat implements DestDir.Lstat.
func (d *DryRunDestDir) Lstat(name string) (os.FileInfo, error) {
	return d.d.Stat(name)
}

// Mkdir implements DestDir.Mkdir.
func (d *DryRunDestDir) Mkdir(name string, perm os.FileMode) error {
	return nil
}

// ReadDir implements DestDir.ReadDir.
func (d *DryRunDestDir) ReadDir(dirname string) ([]os.FileInfo, error) {
	return d.d.ReadDir(dirname)
}

// ReadFile implements DestDir.ReadFile.
func (d *DryRunDestDir) ReadFile(filename string) ([]byte, error) {
	return d.d.ReadFile(filename)
}

// Readlink implements DestDir.Readlink.
func (d *DryRunDestDir) Readlink(name string) (string, error) {
	return d.d.Readlink(name)
}

// RemoveAll implements DestDir.RemoveAll.
func (d *DryRunDestDir) RemoveAll(string) error {
	return nil
}

// Rename implements DestDir.Rename.
func (d *DryRunDestDir) Rename(oldpath, newpath string) error {
	return nil
}

// RunCmd implements DestDir.RunCmd.
func (d *DryRunDestDir) RunCmd(cmd *exec.Cmd) error {
	return nil
}

// Stat implements DestDir.Stat.
func (d *DryRunDestDir) Stat(name string) (os.FileInfo, error) {
	return d.d.Stat(name)
}

// WriteFile implements DestDir.WriteFile.
func (d *DryRunDestDir) WriteFile(string, []byte, os.FileMode, []byte) error {
	return nil
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *DryRunDestDir) WriteSymlink(string, string) error {
	return nil
}
