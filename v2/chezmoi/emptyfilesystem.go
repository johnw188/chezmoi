package chezmoi

import "os"

// An EmptyFileSystemReader represents an empty FileSystem.
type EmptyFileSystemReader struct{}

// Glob implements FileSystem.Glob.
func (*EmptyFileSystemReader) Glob(pattern string) ([]string, error) {
	return nil, nil
}

// Lstat implements FileSystem.Lstat.
func (*EmptyFileSystemReader) Lstat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadDir implements FileSystem.ReadDir.
func (*EmptyFileSystemReader) ReadDir(dirname string) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadFile implements FileSystem.ReadFile.
func (*EmptyFileSystemReader) ReadFile(filename string) ([]byte, error) {
	return nil, os.ErrNotExist
}

// Readlink implements FileSystem.Readlink.
func (*EmptyFileSystemReader) Readlink(name string) (string, error) {
	return "", os.ErrNotExist
}

// Stat implements FileSystem.Stat.
func (*EmptyFileSystemReader) Stat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}
