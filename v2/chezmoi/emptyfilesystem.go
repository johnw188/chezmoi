package chezmoi

import "os"

// An EmptyDestDir represents an empty DestDir.
type EmptyDestDir struct{}

// Glob implements DestDir.Glob.
func (*EmptyDestDir) Glob(pattern string) ([]string, error) {
	return nil, nil
}

// Lstat implements DestDir.Lstat.
func (*EmptyDestDir) Lstat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadDir implements DestDir.ReadDir.
func (*EmptyDestDir) ReadDir(dirname string) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadFile implements DestDir.ReadFile.
func (*EmptyDestDir) ReadFile(filename string) ([]byte, error) {
	return nil, os.ErrNotExist
}

// Readlink implements DestDir.Readlink.
func (*EmptyDestDir) Readlink(name string) (string, error) {
	return "", os.ErrNotExist
}

// Stat implements DestDir.Stat.
func (*EmptyDestDir) Stat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}
