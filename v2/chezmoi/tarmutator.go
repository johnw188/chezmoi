package chezmoi

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"time"
)

type invalidTAROperationError struct {
	operation string
	args      []interface{}
}

func (e *invalidTAROperationError) Error() string {
	return fmt.Sprintf("%s(%v): invalid operation", e.operation, e.args)
}

// A TARMutator is a mutator that writes to a TAR archive.
type TARMutator struct {
	w              *tar.Writer
	headerTemplate tar.Header
	umask          os.FileMode
}

// NewTARMutator returns a new TARMutator that writes a TAR file to w. Commands
// are executed via m.
func NewTARMutator(w io.Writer, headerTemplate tar.Header, umask os.FileMode) *TARMutator {
	return &TARMutator{
		w:              tar.NewWriter(w),
		headerTemplate: headerTemplate,
		umask:          umask,
	}
}

// Chmod implements Mutator.Chmod.
func (m *TARMutator) Chmod(name string, mode os.FileMode) error {
	return &invalidTAROperationError{
		operation: "Chmod",
		args:      []interface{}{name, mode},
	}
}

// Close closes m.
func (m *TARMutator) Close() error {
	return m.w.Close()
}

// IdempotentCmdOutput implements Mutator.IdempotentCmdOutput.
func (m *TARMutator) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// Lstat implements Mutator.Lstat.
func (m *TARMutator) Lstat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// Mkdir implements Mutator.Mkdir.
func (m *TARMutator) Mkdir(name string, perm os.FileMode) error {
	header := m.headerTemplate
	header.Typeflag = tar.TypeDir
	header.Name = name
	header.Mode = int64(perm &^ m.umask)
	return m.w.WriteHeader(&header)
}

// ReadDir implements Mutator.ReadDir.
func (m *TARMutator) ReadDir(dirname string) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// ReadFile implements Mutator.ReadFile.
func (m *TARMutator) ReadFile(filename string) ([]byte, error) {
	return nil, os.ErrNotExist
}

// Readlink implements Mutator.Readlink.
func (m *TARMutator) Readlink(name string) (string, error) {
	return "", os.ErrNotExist
}

// RemoveAll implements Mutator.RemoveAll.
func (m *TARMutator) RemoveAll(name string) error {
	return nil
}

// Rename implements Mutator.Rename.
func (m *TARMutator) Rename(oldpath, newpath string) error {
	return os.ErrNotExist
}

// RunCmd implements Mutator.RunCmd.
func (m *TARMutator) RunCmd(cmd *exec.Cmd) error {
	// FIXME need to work out what to do with scripts
	return nil
}

// Stat implements Mutator.Stat.
func (m *TARMutator) Stat(name string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

// WriteFile implements Mutator.WriteFile.
func (m *TARMutator) WriteFile(filename string, data []byte, perm os.FileMode, currData []byte) error {
	header := m.headerTemplate
	header.Typeflag = tar.TypeReg
	header.Name = filename
	header.Size = int64(len(data))
	header.Mode = int64(perm &^ m.umask)
	if err := m.w.WriteHeader(&header); err != nil {
		return err
	}
	_, err := m.w.Write(data)
	return err
}

// WriteSymlink implements Mutator.WriteSymlink.
func (m *TARMutator) WriteSymlink(oldname, newname string) error {
	header := m.headerTemplate
	header.Typeflag = tar.TypeSymlink
	header.Name = newname
	header.Linkname = oldname
	return m.w.WriteHeader(&header)
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
