package chezmoi

import (
	"os"
	"os/exec"
)

// DryRunMutator is an Mutator that reads from, but does not write, to a wrapped
// Mutator.
type DryRunMutator struct {
	m Mutator
}

// NewDryRunMutator returns a new DryRunMutator that wraps m.
func NewDryRunMutator(m Mutator) *DryRunMutator {
	return &DryRunMutator{
		m: m,
	}
}

// Chmod implements Mutator.Chmod.
func (m *DryRunMutator) Chmod(name string, mode os.FileMode) error {
	return nil
}

// IdempotentCmdOutput implements Mutator.IdempotentCmdOutput.
func (m *DryRunMutator) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return m.m.IdempotentCmdOutput(cmd)
}

// Lstat implements Mutator.Lstat.
func (m *DryRunMutator) Lstat(name string) (os.FileInfo, error) {
	return m.m.Stat(name)
}

// Mkdir implements Mutator.Mkdir.
func (m *DryRunMutator) Mkdir(name string, perm os.FileMode) error {
	return nil
}

// ReadDir implements Mutator.ReadDir.
func (m *DryRunMutator) ReadDir(dirname string) ([]os.FileInfo, error) {
	return m.m.ReadDir(dirname)
}

// ReadFile implements Mutator.ReadFile.
func (m *DryRunMutator) ReadFile(filename string) ([]byte, error) {
	return m.m.ReadFile(filename)
}

// Readlink implements Mutator.Readlink.
func (m *DryRunMutator) Readlink(name string) (string, error) {
	return m.m.Readlink(name)
}

// RemoveAll implements Mutator.RemoveAll.
func (m *DryRunMutator) RemoveAll(string) error {
	return nil
}

// Rename implements Mutator.Rename.
func (m *DryRunMutator) Rename(oldpath, newpath string) error {
	return nil
}

// RunCmd implements Mutator.RunCmd.
func (m *DryRunMutator) RunCmd(cmd *exec.Cmd) error {
	return nil
}

// Stat implements Mutator.Stat.
func (m *DryRunMutator) Stat(name string) (os.FileInfo, error) {
	return m.m.Stat(name)
}

// WriteFile implements Mutator.WriteFile.
func (m *DryRunMutator) WriteFile(string, []byte, os.FileMode, []byte) error {
	return nil
}

// WriteSymlink implements Mutator.WriteSymlink.
func (m *DryRunMutator) WriteSymlink(string, string) error {
	return nil
}
