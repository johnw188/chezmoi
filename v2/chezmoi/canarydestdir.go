package chezmoi

import (
	"os"
	"os/exec"
)

// An CanaryDestDir wraps a DestDir and records if any of its mutating methods
// are called.
type CanaryDestDir struct {
	d       DestDir
	mutated bool
}

// NewCanaryDestDir returns a new CanaryDestDir.
func NewCanaryDestDir(d DestDir) *CanaryDestDir {
	return &CanaryDestDir{
		d:       d,
		mutated: false,
	}
}

// Chmod implements DestDir.Chmod.
func (d *CanaryDestDir) Chmod(name string, mode os.FileMode) error {
	d.mutated = true
	return d.d.Chmod(name, mode)
}

// Glob implements DestDir.Glob.
func (d *CanaryDestDir) Glob(pattern string) ([]string, error) {
	return d.d.Glob(pattern)
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *CanaryDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return d.d.IdempotentCmdOutput(cmd)
}

// Mkdir implements DestDir.Mkdir.
func (d *CanaryDestDir) Mkdir(name string, perm os.FileMode) error {
	d.mutated = true
	return d.d.Mkdir(name, perm)
}

// Lstat implements DestDir.Lstat.
func (d *CanaryDestDir) Lstat(path string) (os.FileInfo, error) {
	return d.d.Lstat(path)
}

// Mutated returns true if any of its mutating methods have been called.
func (d *CanaryDestDir) Mutated() bool {
	return d.mutated
}

// ReadDir implements DestDir.ReadDir.
func (d *CanaryDestDir) ReadDir(dirname string) ([]os.FileInfo, error) {
	return d.d.ReadDir(dirname)
}

// ReadFile implements DestDir.ReadFile.
func (d *CanaryDestDir) ReadFile(filename string) ([]byte, error) {
	return d.d.ReadFile(filename)
}

// Readlink implements DestDir.Readlink.
func (d *CanaryDestDir) Readlink(name string) (string, error) {
	return d.d.Readlink(name)
}

// RemoveAll implements DestDir.RemoveAll.
func (d *CanaryDestDir) RemoveAll(name string) error {
	d.mutated = true
	return d.d.RemoveAll(name)
}

// Rename implements DestDir.Rename.
func (d *CanaryDestDir) Rename(oldpath, newpath string) error {
	d.mutated = true
	return d.d.Rename(oldpath, newpath)
}

// RunCmd implements DestDir.RunCmd.
func (d *CanaryDestDir) RunCmd(cmd *exec.Cmd) error {
	d.mutated = true
	return d.d.RunCmd(cmd)
}

// Stat implements DestDir.Stat.
func (d *CanaryDestDir) Stat(path string) (os.FileInfo, error) {
	return d.d.Stat(path)
}

// WriteFile implements DestDir.WriteFile.
func (d *CanaryDestDir) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	d.mutated = true
	return d.d.WriteFile(name, data, perm, currData)
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *CanaryDestDir) WriteSymlink(oldname, newname string) error {
	d.mutated = true
	return d.d.WriteSymlink(oldname, newname)
}
