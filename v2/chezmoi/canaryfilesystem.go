package chezmoi

import (
	"os"
	"os/exec"
)

// An CanaryFileSystem wraps a FileSystem and records if any of its mutating
// methods are called.
type CanaryFileSystem struct {
	fs      FileSystem
	mutated bool
}

// NewCanaryFileSystem returns a new CanaryFileSystem.
func NewCanaryFileSystem(fs FileSystem) *CanaryFileSystem {
	return &CanaryFileSystem{
		fs:      fs,
		mutated: false,
	}
}

// Chmod implements FileSystem.Chmod.
func (fs *CanaryFileSystem) Chmod(name string, mode os.FileMode) error {
	fs.mutated = true
	return fs.fs.Chmod(name, mode)
}

// Glob implements FileSystem.Glob.
func (fs *CanaryFileSystem) Glob(pattern string) ([]string, error) {
	return fs.fs.Glob(pattern)
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *CanaryFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return fs.fs.IdempotentCmdOutput(cmd)
}

// Mkdir implements FileSystem.Mkdir.
func (d *CanaryFileSystem) Mkdir(name string, perm os.FileMode) error {
	d.mutated = true
	return d.fs.Mkdir(name, perm)
}

// Lstat implements FileSystem.Lstat.
func (fs *CanaryFileSystem) Lstat(path string) (os.FileInfo, error) {
	return fs.fs.Lstat(path)
}

// Mutated returns true if any of its mutating methods have been called.
func (fs *CanaryFileSystem) Mutated() bool {
	return fs.mutated
}

// ReadDir implements FileSystem.ReadDir.
func (fs *CanaryFileSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fs.fs.ReadDir(dirname)
}

// ReadFile implements FileSystem.ReadFile.
func (fs *CanaryFileSystem) ReadFile(filename string) ([]byte, error) {
	return fs.fs.ReadFile(filename)
}

// Readlink implements FileSystem.Readlink.
func (fs *CanaryFileSystem) Readlink(name string) (string, error) {
	return fs.fs.Readlink(name)
}

// RemoveAll implements FileSystem.RemoveAll.
func (fs *CanaryFileSystem) RemoveAll(name string) error {
	fs.mutated = true
	return fs.fs.RemoveAll(name)
}

// Rename implements FileSystem.Rename.
func (fs *CanaryFileSystem) Rename(oldpath, newpath string) error {
	fs.mutated = true
	return fs.fs.Rename(oldpath, newpath)
}

// RunCmd implements FileSystem.RunCmd.
func (fs *CanaryFileSystem) RunCmd(cmd *exec.Cmd) error {
	fs.mutated = true
	return fs.fs.RunCmd(cmd)
}

// Stat implements FileSystem.Stat.
func (fs *CanaryFileSystem) Stat(path string) (os.FileInfo, error) {
	return fs.fs.Stat(path)
}

// WriteFile implements FileSystem.WriteFile.
func (fs *CanaryFileSystem) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	fs.mutated = true
	return fs.fs.WriteFile(name, data, perm, currData)
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *CanaryFileSystem) WriteSymlink(oldname, newname string) error {
	fs.mutated = true
	return fs.fs.WriteSymlink(oldname, newname)
}
