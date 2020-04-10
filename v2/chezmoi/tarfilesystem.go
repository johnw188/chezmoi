package chezmoi

import (
	"archive/tar"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"time"
)

// A TARFileSystem is a FileSystem that writes to a TAR archive.
type TARFileSystem struct {
	*EmptyFileSystemReader
	w              *tar.Writer
	headerTemplate tar.Header
	umask          os.FileMode
}

// NewTARFileSystem returns a new TARFileSystem that writes a TAR file to w.
func NewTARFileSystem(w io.Writer, headerTemplate tar.Header, umask os.FileMode) *TARFileSystem {
	return &TARFileSystem{
		w:              tar.NewWriter(w),
		headerTemplate: headerTemplate,
		umask:          umask,
	}
}

// Chmod implements FileSystem.Chmod.
func (fs *TARFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.ErrPermission
}

// Close closes m.
func (fs *TARFileSystem) Close() error {
	return fs.w.Close()
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *TARFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// Mkdir implements FileSystem.Mkdir.
func (fs *TARFileSystem) Mkdir(name string, perm os.FileMode) error {
	header := fs.headerTemplate
	header.Typeflag = tar.TypeDir
	header.Name = name
	header.Mode = int64(perm &^ fs.umask)
	return fs.w.WriteHeader(&header)
}

// RemoveAll implements FileSystem.RemoveAll.
func (fs *TARFileSystem) RemoveAll(name string) error {
	return os.ErrPermission
}

// Rename implements FileSystem.Rename.
func (fs *TARFileSystem) Rename(oldpath, newpath string) error {
	return os.ErrPermission
}

// RunCmd implements FileSystem.RunCmd.
func (fs *TARFileSystem) RunCmd(cmd *exec.Cmd) error {
	// FIXME need to work out what to do with scripts
	return nil
}

// WriteFile implements FileSystem.WriteFile.
func (fs *TARFileSystem) WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error {
	header := fs.headerTemplate
	header.Typeflag = tar.TypeReg
	header.Name = filename
	header.Size = int64(len(data))
	header.Mode = int64(perm &^ fs.umask)
	if err := fs.w.WriteHeader(&header); err != nil {
		return err
	}
	_, err := fs.w.Write(data)
	return err
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *TARFileSystem) WriteSymlink(oldname, newname string) error {
	header := fs.headerTemplate
	header.Typeflag = tar.TypeSymlink
	header.Name = newname
	header.Linkname = oldname
	return fs.w.WriteHeader(&header)
}

// TARHeaderTemplate returns a tar.Header template populated with the current
// user and time.
func TARHeaderTemplate() tar.Header {
	// Attempt to lookup the current user. Ignore errors because the default
	// zero values are reasonable.
	var (
		uid   int
		gid   int
		Uname string
		Gname string
	)
	if currentUser, err := user.Current(); err == nil {
		uid, _ = strconv.Atoi(currentUser.Uid)
		gid, _ = strconv.Atoi(currentUser.Gid)
		Uname = currentUser.Username
		if group, err := user.LookupGroupId(currentUser.Gid); err == nil {
			Gname = group.Name
		}
	}

	now := time.Now()
	return tar.Header{
		Uid:        uid,
		Gid:        gid,
		Uname:      Uname,
		Gname:      Gname,
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	}
}
