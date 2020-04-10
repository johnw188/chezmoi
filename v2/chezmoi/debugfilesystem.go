package chezmoi

import (
	"log"
	"os"
	"os/exec"
	"time"
)

// A DebugFileSystem wraps a FileSystem and logs all of the actions it executes.
type DebugFileSystem struct {
	fs FileSystem
}

// NewDebugFileSystem returns a new DebugFileSystem.
func NewDebugFileSystem(fs FileSystem) *DebugFileSystem {
	return &DebugFileSystem{
		fs: fs,
	}
}

// Chmod implements FileSystem.Chmod.
func (fs *DebugFileSystem) Chmod(name string, mode os.FileMode) error {
	return Debugf("Chmod(%q, 0o%o)", []interface{}{name, mode}, func() error {
		return fs.fs.Chmod(name, mode)
	})
}

// Glob implements FileSystem.Glob.
func (fs *DebugFileSystem) Glob(name string) ([]string, error) {
	var matches []string
	err := Debugf("Glob(%q)", []interface{}{name}, func() error {
		var err error
		matches, err = fs.fs.Glob(name)
		return err
	})
	return matches, err
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *DebugFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	var output []byte
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	err := Debugf("IdempotentCmdOutput(%q)", []interface{}{cmdStr}, func() error {
		var err error
		output, err = fs.fs.IdempotentCmdOutput(cmd)
		return err
	})
	return output, err
}

// Lstat implements FileSystem.Lstat.
func (fs *DebugFileSystem) Lstat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := Debugf("Lstat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = fs.fs.Lstat(name)
		return err
	})
	return info, err
}

// Mkdir implements FileSystem.Mkdir.
func (fs *DebugFileSystem) Mkdir(name string, perm os.FileMode) error {
	return Debugf("Mkdir(%q, 0o%o)", []interface{}{name, perm}, func() error {
		return fs.fs.Mkdir(name, perm)
	})
}

// ReadDir implements FileSystem.ReadDir.
func (fs *DebugFileSystem) ReadDir(name string) ([]os.FileInfo, error) {
	var infos []os.FileInfo
	err := Debugf("ReadDir(%q)", []interface{}{name}, func() error {
		var err error
		infos, err = fs.fs.ReadDir(name)
		return err
	})
	return infos, err
}

// ReadFile implements FileSystem.ReadFile.
func (fs *DebugFileSystem) ReadFile(filename string) ([]byte, error) {
	var data []byte
	err := Debugf("ReadFile(%q)", []interface{}{filename}, func() error {
		var err error
		data, err = fs.fs.ReadFile(filename)
		return err
	})
	return data, err
}

// Readlink implements FileSystem.Readlink.
func (fs *DebugFileSystem) Readlink(name string) (string, error) {
	var linkname string
	err := Debugf("Readlink(%q)", []interface{}{name}, func() error {
		var err error
		linkname, err = fs.fs.Readlink(name)
		return err
	})
	return linkname, err
}

// RemoveAll implements FileSystem.RemoveAll.
func (fs *DebugFileSystem) RemoveAll(name string) error {
	return Debugf("RemoveAll(%q)", []interface{}{name}, func() error {
		return fs.fs.RemoveAll(name)
	})
}

// Rename implements FileSystem.Rename.
func (fs *DebugFileSystem) Rename(oldpath, newpath string) error {
	return Debugf("Rename(%q, %q)", []interface{}{oldpath, newpath}, func() error {
		return fs.Rename(oldpath, newpath)
	})
}

// RunCmd implements FileSystem.RunCmd.
func (fs *DebugFileSystem) RunCmd(cmd *exec.Cmd) error {
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	return Debugf("Run(%q)", []interface{}{cmdStr}, func() error {
		return fs.fs.RunCmd(cmd)
	})
}

// Stat implements FileSystem.Stat.
func (fs *DebugFileSystem) Stat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := Debugf("Stat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = fs.fs.Stat(name)
		return err
	})
	return info, err
}

// WriteFile implements FileSystem.WriteFile.
func (fs *DebugFileSystem) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	return Debugf("WriteFile(%q, _, 0%o, _)", []interface{}{name, perm}, func() error {
		return fs.fs.WriteFile(name, data, perm, currData)
	})
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *DebugFileSystem) WriteSymlink(oldname, newname string) error {
	return Debugf("WriteSymlink(%q, %q)", []interface{}{oldname, newname}, func() error {
		return fs.fs.WriteSymlink(oldname, newname)
	})
}

// Debugf logs debugging information about calling f.
func Debugf(format string, args []interface{}, f func() error) error {
	errChan := make(chan error)
	start := time.Now()
	go func(errChan chan<- error) {
		errChan <- f()
	}(errChan)
	select {
	case err := <-errChan:
		if err == nil {
			log.Printf(format+" (%s)", append(args, time.Since(start))...)
		} else {
			log.Printf(format+" == %v (%s)", append(args, err, time.Since(start))...)
		}
		return err
	case <-time.After(1 * time.Second):
		log.Printf(format, args...)
		err := <-errChan
		if err == nil {
			log.Printf(format+" (%s)", append(args, time.Since(start))...)
		} else {
			log.Printf(format+" == %v (%s)", append(args, err, time.Since(start))...)
		}
		return err
	}
}
