package chezmoi

// FIXME do we need Stat?
// FIXME do we need a more specific FileReader interface with just ReadFile?

import (
	"os"
	"os/exec"
)

// A FileSystemReader reads from a file system.
type FileSystemReader interface {
	Glob(pattern string) ([]string, error)
	Lstat(filename string) (os.FileInfo, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	Readlink(name string) (string, error)
	Stat(name string) (os.FileInfo, error)
}

// A FileSystem writes to a file system.
type FileSystem interface {
	FileSystemReader
	Chmod(name string, mode os.FileMode) error
	IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error)
	Mkdir(name string, perm os.FileMode) error
	RemoveAll(name string) error
	Rename(oldpath, newpath string) error
	RunCmd(cmd *exec.Cmd) error
	WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error
	WriteSymlink(oldname, newname string) error
}
