package chezmoi

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/diff"
)

// A VerboseDestDir wraps a DestDir and logs all of the actions it executes and
// any errors as pseudo shell commands.
type VerboseDestDir struct {
	d               DestDir
	w               io.Writer
	colored         bool
	maxDiffDataSize int
}

// NewVerboseDestDir returns a new VerboseDestDir.
func NewVerboseDestDir(w io.Writer, m DestDir, colored bool, maxDiffDataSize int) *VerboseDestDir {
	return &VerboseDestDir{
		d:               m,
		w:               w,
		colored:         colored,
		maxDiffDataSize: maxDiffDataSize,
	}
}

// Chmod implements DestDir.Chmod.
func (d *VerboseDestDir) Chmod(name string, mode os.FileMode) error {
	action := fmt.Sprintf("chmod %o %s", mode, MaybeShellQuote(name))
	err := d.d.Chmod(name, mode)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// IdempotentCmdOutput implements DestDir.IdempotentCmdOutput.
func (d *VerboseDestDir) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	action := cmdString(cmd)
	output, err := d.d.IdempotentCmdOutput(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return output, err
}

// Lstat implements DestDir.Lstat.
func (d *VerboseDestDir) Lstat(name string) (os.FileInfo, error) {
	return d.d.Lstat(name)
}

// ReadDir implements DestDir.ReadDir.
func (d *VerboseDestDir) ReadDir(dirname string) ([]os.FileInfo, error) {
	return d.d.ReadDir(dirname)
}

// ReadFile implements DestDir.ReadFile.
func (d *VerboseDestDir) ReadFile(filename string) ([]byte, error) {
	return d.d.ReadFile(filename)
}

// Readlink implements DestDir.Readlink.
func (d *VerboseDestDir) Readlink(name string) (string, error) {
	return d.d.Readlink(name)
}

// Mkdir implements DestDir.Mkdir.
func (d *VerboseDestDir) Mkdir(name string, perm os.FileMode) error {
	action := fmt.Sprintf("mkdir -m %o %s", perm, MaybeShellQuote(name))
	err := d.d.Mkdir(name, perm)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// RemoveAll implements DestDir.RemoveAll.
func (d *VerboseDestDir) RemoveAll(name string) error {
	action := fmt.Sprintf("rm -rf %s", MaybeShellQuote(name))
	err := d.d.RemoveAll(name)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// Rename implements DestDir.Rename.
func (d *VerboseDestDir) Rename(oldpath, newpath string) error {
	action := fmt.Sprintf("mv %s %s", MaybeShellQuote(oldpath), MaybeShellQuote(newpath))
	err := d.d.Rename(oldpath, newpath)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// RunCmd implements DestDir.RunCmd.
func (d *VerboseDestDir) RunCmd(cmd *exec.Cmd) error {
	action := cmdString(cmd)
	err := d.d.RunCmd(cmd)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// Stat implements DestDir.Stat.
func (d *VerboseDestDir) Stat(name string) (os.FileInfo, error) {
	return d.d.Stat(name)
}

// WriteFile implements DestDir.WriteFile.
func (d *VerboseDestDir) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	action := fmt.Sprintf("install -m %o /dev/null %s", perm, MaybeShellQuote(name))
	err := d.d.WriteFile(name, data, perm, currData)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
		// Don't print diffs if either file is binary.
		if isBinary(currData) || isBinary(data) {
			return nil
		}
		// Don't print diffs if either file is too large.
		if d.maxDiffDataSize != 0 {
			if len(currData) > d.maxDiffDataSize || len(data) > d.maxDiffDataSize {
				return nil
			}
		}
		aLines, err := splitLines(currData)
		if err != nil {
			return err
		}
		bLines, err := splitLines(data)
		if err != nil {
			return err
		}
		ab := diff.Strings(aLines, bLines)
		e := diff.Myers(context.Background(), ab).WithContextSize(3)
		opts := []diff.WriteOpt{
			diff.Names(
				path.Join("a", name),
				path.Join("b", name),
			),
		}
		if d.colored {
			opts = append(opts, diff.TerminalColor())
		}
		if _, err := e.WriteUnified(d.w, ab, opts...); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// WriteSymlink implements DestDir.WriteSymlink.
func (d *VerboseDestDir) WriteSymlink(oldname, newname string) error {
	action := fmt.Sprintf("ln -sf %s %s", MaybeShellQuote(oldname), MaybeShellQuote(newname))
	err := d.d.WriteSymlink(oldname, newname)
	if err == nil {
		_, _ = fmt.Fprintln(d.w, action)
	} else {
		_, _ = fmt.Fprintf(d.w, "%s: %v\n", action, err)
	}
	return err
}

// cmdString returns a string representation of cmd.
func cmdString(cmd *exec.Cmd) string {
	s := ShellQuoteArgs(append([]string{cmd.Path}, cmd.Args[1:]...))
	if cmd.Dir == "" {
		return s
	}
	return fmt.Sprintf("( cd %s && %s )", MaybeShellQuote(cmd.Dir), s)
}

func isBinary(data []byte) bool {
	return len(data) != 0 && !strings.HasPrefix(http.DetectContentType(data), "text/")
}

func splitLines(data []byte) ([]string, error) {
	var lines []string
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	return lines, s.Err()
}
