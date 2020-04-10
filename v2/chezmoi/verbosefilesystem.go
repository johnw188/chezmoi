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

// A VerboseFileSystem wraps a FileSystem and logs all of the actions it executes and
// any errors as pseudo shell commands.
type VerboseFileSystem struct {
	fs              FileSystem
	w               io.Writer
	colored         bool
	maxDiffDataSize int
}

// NewVerboseFileSystem returns a new VerboseFileSystem.
func NewVerboseFileSystem(w io.Writer, m FileSystem, colored bool, maxDiffDataSize int) *VerboseFileSystem {
	return &VerboseFileSystem{
		fs:              m,
		w:               w,
		colored:         colored,
		maxDiffDataSize: maxDiffDataSize,
	}
}

// Chmod implements FileSystem.Chmod.
func (fs *VerboseFileSystem) Chmod(name string, mode os.FileMode) error {
	action := fmt.Sprintf("chmod %o %s", mode, MaybeShellQuote(name))
	err := fs.fs.Chmod(name, mode)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// Glob implements FileSystem.Glob.
func (fs *VerboseFileSystem) Glob(pattern string) ([]string, error) {
	return fs.fs.Glob(pattern)
}

// IdempotentCmdOutput implements FileSystem.IdempotentCmdOutput.
func (fs *VerboseFileSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	action := cmdString(cmd)
	output, err := fs.fs.IdempotentCmdOutput(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return output, err
}

// Lstat implements FileSystem.Lstat.
func (fs *VerboseFileSystem) Lstat(name string) (os.FileInfo, error) {
	return fs.fs.Lstat(name)
}

// ReadDir implements FileSystem.ReadDir.
func (fs *VerboseFileSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fs.fs.ReadDir(dirname)
}

// ReadFile implements FileSystem.ReadFile.
func (fs *VerboseFileSystem) ReadFile(filename string) ([]byte, error) {
	return fs.fs.ReadFile(filename)
}

// Readlink implements FileSystem.Readlink.
func (fs *VerboseFileSystem) Readlink(name string) (string, error) {
	return fs.fs.Readlink(name)
}

// Mkdir implements FileSystem.Mkdir.
func (fs *VerboseFileSystem) Mkdir(name string, perm os.FileMode) error {
	action := fmt.Sprintf("mkdir -m %o %s", perm, MaybeShellQuote(name))
	err := fs.fs.Mkdir(name, perm)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// RemoveAll implements FileSystem.RemoveAll.
func (fs *VerboseFileSystem) RemoveAll(name string) error {
	action := fmt.Sprintf("rm -rf %s", MaybeShellQuote(name))
	err := fs.fs.RemoveAll(name)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// Rename implements FileSystem.Rename.
func (fs *VerboseFileSystem) Rename(oldpath, newpath string) error {
	action := fmt.Sprintf("mv %s %s", MaybeShellQuote(oldpath), MaybeShellQuote(newpath))
	err := fs.fs.Rename(oldpath, newpath)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// RunCmd implements FileSystem.RunCmd.
func (fs *VerboseFileSystem) RunCmd(cmd *exec.Cmd) error {
	action := cmdString(cmd)
	err := fs.fs.RunCmd(cmd)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// Stat implements FileSystem.Stat.
func (fs *VerboseFileSystem) Stat(name string) (os.FileInfo, error) {
	return fs.fs.Stat(name)
}

// WriteFile implements FileSystem.WriteFile.
func (fs *VerboseFileSystem) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	action := fmt.Sprintf("install -m %o /dev/null %s", perm, MaybeShellQuote(name))
	err := fs.fs.WriteFile(name, data, perm, currData)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
		// Don't print diffs if either file is binary.
		if isBinary(currData) || isBinary(data) {
			return nil
		}
		// Don't print diffs if either file is too large.
		if fs.maxDiffDataSize != 0 {
			if len(currData) > fs.maxDiffDataSize || len(data) > fs.maxDiffDataSize {
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
		if fs.colored {
			opts = append(opts, diff.TerminalColor())
		}
		if _, err := e.WriteUnified(fs.w, ab, opts...); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
	}
	return err
}

// WriteSymlink implements FileSystem.WriteSymlink.
func (fs *VerboseFileSystem) WriteSymlink(oldname, newname string) error {
	action := fmt.Sprintf("ln -sf %s %s", MaybeShellQuote(oldname), MaybeShellQuote(newname))
	err := fs.fs.WriteSymlink(oldname, newname)
	if err == nil {
		_, _ = fmt.Fprintln(fs.w, action)
	} else {
		_, _ = fmt.Fprintf(fs.w, "%s: %v\n", action, err)
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
