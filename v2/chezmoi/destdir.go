package chezmoi

import "os"

// A DestDir is destination directory.
type DestDir interface {
	Lstat(filename string) (os.FileInfo, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	ReadFile(filename string) ([]byte, error)
}

// A DestDirEmpty represents an empty destination directory.
type DestDirEmpty struct{}

// Lstat implements os.Lstat.
func (*DestDirEmpty) Lstat(filename string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadDir implements ioutil.ReadDir.
func (*DestDirEmpty) ReadDir(dirname string) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadFile implements ioutil.ReadFile.
func (*DestDirEmpty) ReadFile(filename string) ([]byte, error) {
	return nil, os.ErrNotExist
}
