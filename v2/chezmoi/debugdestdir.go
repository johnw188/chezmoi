package chezmoi

import (
	"log"
	"os"
	"os/exec"
	"time"
)

// A DebugDestDir wraps a DestDir and logs all of the actions it executes.
type DebugDestDir struct {
	d DestDir
}

// NewDebugDestDir returns a new DebugDestDir.
func NewDebugDestDir(d DestDir) *DebugDestDir {
	return &DebugDestDir{
		d: d,
	}
}

// Chmod implements DestDir.Chmod.
func (d *DebugDestDir) Chmod(name string, mode os.FileMode) error {
	return Debugf("Chmod(%q, 0o%o)", []interface{}{name, mode}, func() error {
		return d.d.Chmod(name, mode)
	})
}

// Glob implements DestDir.Glob.
func (d *DebugDestDir) Glob(name string) ([]string, error) {
	var matches []string
	err := Debugf("Glob(%q)", []interface{}{name}, func() error {
		var err error
		matches, err = d.d.Glob(name)
		return err
	})
	return matches, err
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *DebugDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	var output []byte
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	err := Debugf("IdempotentCmdOutput(%q)", []interface{}{cmdStr}, func() error {
		var err error
		output, err = d.d.IdempotentCmdOutput(cmd)
		return err
	})
	return output, err
}

// Lstat implements DestDir.Lstat.
func (d *DebugDestDir) Lstat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := Debugf("Lstat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = d.d.Lstat(name)
		return err
	})
	return info, err
}

// Mkdir implements DestDir.Mkdir.
func (d *DebugDestDir) Mkdir(name string, perm os.FileMode) error {
	return Debugf("Mkdir(%q, 0o%o)", []interface{}{name, perm}, func() error {
		return d.d.Mkdir(name, perm)
	})
}

// ReadDir implements DestDir.ReadDir.
func (d *DebugDestDir) ReadDir(name string) ([]os.FileInfo, error) {
	var infos []os.FileInfo
	err := Debugf("ReadDir(%q)", []interface{}{name}, func() error {
		var err error
		infos, err = d.d.ReadDir(name)
		return err
	})
	return infos, err
}

// ReadFile implements DestDir.ReadFile.
func (d *DebugDestDir) ReadFile(filename string) ([]byte, error) {
	var data []byte
	err := Debugf("ReadFile(%q)", []interface{}{filename}, func() error {
		var err error
		data, err = d.d.ReadFile(filename)
		return err
	})
	return data, err
}

// Readlink implements DestDir.Readlink.
func (d *DebugDestDir) Readlink(name string) (string, error) {
	var linkname string
	err := Debugf("Readlink(%q)", []interface{}{name}, func() error {
		var err error
		linkname, err = d.d.Readlink(name)
		return err
	})
	return linkname, err
}

// RemoveAll implements DestDir.RemoveAll.
func (d *DebugDestDir) RemoveAll(name string) error {
	return Debugf("RemoveAll(%q)", []interface{}{name}, func() error {
		return d.d.RemoveAll(name)
	})
}

// Rename implements DestDir.Rename.
func (d *DebugDestDir) Rename(oldpath, newpath string) error {
	return Debugf("Rename(%q, %q)", []interface{}{oldpath, newpath}, func() error {
		return d.Rename(oldpath, newpath)
	})
}

// RunCmd implements DestDir.RunCmd.
func (d *DebugDestDir) RunCmd(cmd *exec.Cmd) error {
	cmdStr := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	return Debugf("Run(%q)", []interface{}{cmdStr}, func() error {
		return d.d.RunCmd(cmd)
	})
}

// Stat implements DestDir.Stat.
func (d *DebugDestDir) Stat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := Debugf("Stat(%q)", []interface{}{name}, func() error {
		var err error
		info, err = d.d.Stat(name)
		return err
	})
	return info, err
}

// WriteFile implements DestDir.WriteFile.
func (d *DebugDestDir) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	return Debugf("WriteFile(%q, _, 0%o, _)", []interface{}{name, perm}, func() error {
		return d.d.WriteFile(name, data, perm, currData)
	})
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *DebugDestDir) WriteSymlink(oldname, newname string) error {
	return Debugf("WriteSymlink(%q, %q)", []interface{}{oldname, newname}, func() error {
		return d.d.WriteSymlink(oldname, newname)
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
