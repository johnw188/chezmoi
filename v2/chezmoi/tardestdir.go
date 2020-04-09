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

// A TARDestDir is a DestDir that writes to a TAR archive.
type TARDestDir struct {
	*EmptyDestDir
	w              *tar.Writer
	headerTemplate tar.Header
	umask          os.FileMode
}

// NewTARDestDir returns a new TARDestDir that writes a TAR file to w.
func NewTARDestDir(w io.Writer, headerTemplate tar.Header, umask os.FileMode) *TARDestDir {
	return &TARDestDir{
		w:              tar.NewWriter(w),
		headerTemplate: headerTemplate,
		umask:          umask,
	}
}

// Chmod implements DestDir.Chmod.
func (d *TARDestDir) Chmod(name string, mode os.FileMode) error {
	return os.ErrPermission
}

// Close closes m.
func (d *TARDestDir) Close() error {
	return d.w.Close()
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *TARDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// Mkdir implements DestDir.Mkdir.
func (d *TARDestDir) Mkdir(name string, perm os.FileMode) error {
	header := d.headerTemplate
	header.Typeflag = tar.TypeDir
	header.Name = name
	header.Mode = int64(perm &^ d.umask)
	return d.w.WriteHeader(&header)
}

// RemoveAll implements DestDir.RemoveAll.
func (d *TARDestDir) RemoveAll(name string) error {
	return os.ErrPermission
}

// Rename implements DestDir.Rename.
func (d *TARDestDir) Rename(oldpath, newpath string) error {
	return os.ErrPermission
}

// RunCmd implements DestDir.RunCmd.
func (d *TARDestDir) RunCmd(cmd *exec.Cmd) error {
	// FIXME need to work out what to do with scripts
	return nil
}

// WriteFile implements DestDir.WriteFile.
func (d *TARDestDir) WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error {
	header := d.headerTemplate
	header.Typeflag = tar.TypeReg
	header.Name = filename
	header.Size = int64(len(data))
	header.Mode = int64(perm &^ d.umask)
	if err := d.w.WriteHeader(&header); err != nil {
		return err
	}
	_, err := d.w.Write(data)
	return err
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *TARDestDir) WriteSymlink(oldname, newname string) error {
	header := d.headerTemplate
	header.Typeflag = tar.TypeSymlink
	header.Name = newname
	header.Linkname = oldname
	return d.w.WriteHeader(&header)
}

func getHeaderTemplate() tar.Header {
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
