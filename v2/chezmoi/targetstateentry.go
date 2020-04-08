package chezmoi

// FIXME UmaskMutator and add PermEqual method?

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

// A TargetStateEntry represents the state of an entry in the target state.
type TargetStateEntry interface {
	Apply(mutator Mutator, destStateEntry DestStateEntry) error
	Equal(destStateEntry DestStateEntry) (bool, error)
	Evaluate() error
}

// A TargetStateAbsent represents the absence of an entry in the target state.
type TargetStateAbsent struct{}

// A TargetStateDir represents the state of a directory in the target state.
type TargetStateDir struct {
	perm  os.FileMode
	exact bool
}

// A TargetStateFile represents the state of a file in the target state.
type TargetStateFile struct {
	perm os.FileMode
	*lazyContents
}

// A TargetStateScript represents the state of a script.
// FIXME maybe scripts should be handled specially
type TargetStateScript struct {
	name string
	*lazyContents
}

// A TargetStateSymlink represents the state of a symlink in the target state.
type TargetStateSymlink struct {
	*lazyLinkname
}

// Apply updates destStateEntry to match t.
func (t *TargetStateAbsent) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	if _, ok := destStateEntry.(*DestStateAbsent); ok {
		return nil
	}
	return mutator.RemoveAll(destStateEntry.Path())
}

// Equal returns true if destStateEntry matches t.
func (t *TargetStateAbsent) Equal(destStateEntry DestStateEntry) (bool, error) {
	_, ok := destStateEntry.(*DestStateAbsent)
	return ok, nil
}

// Evaluate evaluates t.
func (t *TargetStateAbsent) Evaluate() error {
	return nil
}

// Apply updates destStateEntry to match t. It does not recurse.
func (t *TargetStateDir) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	if destStateDir, ok := destStateEntry.(*DestStateDir); ok {
		if destStateDir.perm == t.perm {
			return nil
		}
		return mutator.Chmod(destStateDir.Path(), t.perm)
	}
	if err := destStateEntry.Remove(mutator); err != nil {
		return err
	}
	return mutator.Mkdir(destStateEntry.Path(), t.perm)
}

// Equal returns true if destStateEntry matches t. It does not recurse.
func (t *TargetStateDir) Equal(destStateEntry DestStateEntry) (bool, error) {
	destStateDir, ok := destStateEntry.(*DestStateDir)
	if !ok {
		return false, nil
	}
	return destStateDir.perm == t.perm, nil
}

// Evaluate evaluates t.
func (t *TargetStateDir) Evaluate() error {
	return nil
}

// Apply updates destStateEntry to match t.
func (t *TargetStateFile) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	var destContents []byte
	destIsFileAndPermMatches := false
	if destStateFile, ok := destStateEntry.(*DestStateFile); ok {
		// Compare file contents using only their SHA256 sums. This is so that
		// we can compare last-written states without storing the full contents
		// of each file written.
		destContentsSHA256, err := destStateFile.ContentsSHA256()
		if err != nil {
			return err
		}
		contentsSHA256, err := t.ContentsSHA256()
		if err != nil {
			return err
		}
		if bytes.Equal(destContentsSHA256, contentsSHA256) {
			if destStateFile.perm == t.perm {
				return nil
			}
			return mutator.Chmod(destStateFile.Path(), t.perm)
		}
		destContents, err = destStateFile.Contents()
		if err != nil {
			return err
		}
		destIsFileAndPermMatches = destStateFile.perm == t.perm
	}
	contents, err := t.Contents()
	if err != nil {
		return err
	}
	// If the destination entry is a file and its permissions match the target
	// state then we can rely on mutator.WriteFile to replace the file, possibly
	// atomically. Otherwise we must remove the destination entry before writing
	// the new file. If the destination entry is not a file then it must be
	// removed as mutator.WriteFile will not overwrite non-files. If the
	// destination entry is a file but the permissions do not matchq then we
	// must remove the file first because there is no way atomically update the
	// permissions and the content simultaneously.
	//
	// FIXME update Mutator.WriteFile to truncate, update perms, then write content
	if !destIsFileAndPermMatches {
		if err := destStateEntry.Remove(mutator); err != nil {
			return err
		}
	}
	return mutator.WriteFile(destStateEntry.Path(), contents, t.perm, destContents)
}

// Equal returns true if destStateEntry matches t.
func (t *TargetStateFile) Equal(destStateEntry DestStateEntry) (bool, error) {
	destStateFile, ok := destStateEntry.(*DestStateFile)
	if !ok {
		return false, nil
	}
	if destStateFile.perm != t.perm {
		return false, nil
	}
	destContentsSHA256, err := destStateFile.ContentsSHA256()
	if err != nil {
		return false, err
	}
	contentsSHA256, err := t.ContentsSHA256()
	if err != nil {
		return false, err
	}
	return bytes.Equal(destContentsSHA256, contentsSHA256), nil
}

// Evaluate evaluates t.
func (t *TargetStateFile) Evaluate() error {
	_, err := t.ContentsSHA256()
	return err
}

// Apply does nothing for scripts.
// FIXME maybe this should call Run?
func (t *TargetStateScript) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	return nil
}

// Equal returns true if destStateEntry matches t.
func (t *TargetStateScript) Equal(destStateEntry DestStateEntry) (bool, error) {
	// Scripts are independent of the destination state.
	// FIXME maybe the destination state should store the sha256 sums of executed scripts
	return true, nil
}

// Evaluate evaluates t.
func (t *TargetStateScript) Evaluate() error {
	_, err := t.ContentsSHA256()
	return err
}

// Run runs t.
func (t *TargetStateScript) Run(mutator Mutator) error {
	contents, err := t.Contents()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(contents)) == 0 {
		// Don't execute empty scripts.
		return nil
	}

	// FIXME once_
	// FIXME verbose and dry run -- maybe handled by mutator?

	// Write the temporary script file. Put the randomness at the front of the
	// filename to preserve any file extension for Windows scripts.
	f, err := ioutil.TempFile("", "*."+path.Base(t.name))
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(f.Name())
	}()

	// Make the script private before writing it in case it contains any
	// secrets.
	if err := f.Chmod(0o700); err != nil {
		return err
	}
	if _, err := f.Write(contents); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Run the temporary script file.
	//nolint:gosec
	c := exec.Command(f.Name())
	// c.Dir = path.Join(applyOptions.DestDir, filepath.Dir(s.targetName)) // FIXME
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := mutator.RunCmd(c); err != nil { // FIXME
		return err
	}

	// FIXME record run if once_

	return nil
}

// Apply updates destStateEntry to match t.
func (t *TargetStateSymlink) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	if destStateSymlink, ok := destStateEntry.(*DestStateSymlink); ok {
		destLinkname, err := destStateSymlink.Linkname()
		if err != nil {
			return err
		}
		linkname, err := t.Linkname()
		if err != nil {
			return err
		}
		if destLinkname == linkname {
			return nil
		}
	}
	linkname, err := t.Linkname()
	if err != nil {
		return err
	}
	if err := destStateEntry.Remove(mutator); err != nil {
		return err
	}
	return mutator.WriteSymlink(linkname, destStateEntry.Path())
}

// Equal returns true if destStateEntry matches t.
func (t *TargetStateSymlink) Equal(destStateEntry DestStateEntry) (bool, error) {
	destStateSymlink, ok := destStateEntry.(*DestStateSymlink)
	if !ok {
		return false, nil
	}
	destLinkname, err := destStateSymlink.Linkname()
	if err != nil {
		return false, err
	}
	linkname, err := t.Linkname()
	if err != nil {
		return false, nil
	}
	return destLinkname == linkname, nil
}

// Evaluate evaluates t.
func (t *TargetStateSymlink) Evaluate() error {
	_, err := t.Linkname()
	return err
}
