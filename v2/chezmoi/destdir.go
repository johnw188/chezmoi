package chezmoi

import (
	"os"
	"os/exec"
)

// A DestDirReader reads from a destination directory.
type DestDirReader interface {
	Glob(pattern string) ([]string, error)
	Lstat(filename string) (os.FileInfo, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
	Readlink(name string) (string, error)
	Stat(name string) (os.FileInfo, error)
}

// A DestDir makes changes to a destination directory.
type DestDir interface {
	DestDirReader
	Chmod(name string, mode os.FileMode) error
	IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error)
	Mkdir(name string, perm os.FileMode) error
	RemoveAll(name string) error
	Rename(oldpath, newpath string) error
	RunCmd(cmd *exec.Cmd) error
	WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error
	WriteSymlink(oldname, newname string) error
}
