package chezmoi

import (
	"os"
	"os/exec"
)

// DryRunFileSystem is an FileSystem that reads from, but does not write to, to
// a wrapped FileSystem.
type DryRunFileSystem struct {
	fs FileSystem
}

// NewDryRunFileSystem returns a new DryRunFileSystem that wraps fs.
func NewDryRunFileSystem(fs FileSystem) *DryRunFileSystem {
	return &DryRunFileSystem{
		fs: fs,
	}
}

// Chmod implements FileSystem.Chmod.
func (fs *DryRunFileSystem) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Glob implements FileSystem.Glob.
func (fs *DryRunFileSystem) Glob(pattern string) ([]string, error) {
	return fs.fs.Glob(pattern)
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *DryRunFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return fs.fs.IdempotentCmdOutput(cmd)
}

// Lstat implements FileSystem.Lstat.
func (fs *DryRunFileSystem) Lstat(name string) (os.FileInfo, error) {
	return fs.fs.Stat(name)
}

// Mkdir implements FileSystem.Mkdir.
func (fs *DryRunFileSystem) Mkdir(name string, perm os.FileMode) error {
	return nil
}

// ReadDir implements FileSystem.ReadDir.
func (fs *DryRunFileSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fs.fs.ReadDir(dirname)
}

// ReadFile implements FileSystem.ReadFile.
func (fs *DryRunFileSystem) ReadFile(filename string) ([]byte, error) {
	return fs.fs.ReadFile(filename)
}

// Readlink implements FileSystem.Readlink.
func (fs *DryRunFileSystem) Readlink(name string) (string, error) {
	return fs.fs.Readlink(name)
}

// RemoveAll implements FileSystem.RemoveAll.
func (fs *DryRunFileSystem) RemoveAll(string) error {
	return nil
}

// Rename implements FileSystem.Rename.
func (fs *DryRunFileSystem) Rename(oldpath, newpath string) error {
	return nil
}

// RunCmd implements FileSystem.RunCmd.
func (fs *DryRunFileSystem) RunCmd(cmd *exec.Cmd) error {
	return nil
}

// Stat implements FileSystem.Stat.
func (fs *DryRunFileSystem) Stat(name string) (os.FileInfo, error) {
	return fs.fs.Stat(name)
}

// WriteFile implements FileSystem.WriteFile.
func (fs *DryRunFileSystem) WriteFile(string, []byte, os.FileMode, []byte) error {
	return nil
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *DryRunFileSystem) WriteSymlink(string, string) error {
	return nil
}
