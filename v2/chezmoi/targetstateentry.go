package chezmoi

import (
	"bytes"
	"os"
)

// A TargetStateEntry represents the state of an entry in the target state.
type TargetStateEntry interface {
	Apply(mutator Mutator, destStateEntry DestStateEntry) error
	Equal(destStateEntry DestStateEntry) (bool, error)
}

// A TargetStateAbsent represents the absence of an entry in the target state.
type TargetStateAbsent struct{}

// A TargetStateDir represents the state of a directory in the target state.
type TargetStateDir struct {
	perm os.FileMode
}

// A TargetStateFile represents the state of a file in the target state.
type TargetStateFile struct {
	perm os.FileMode
	*LazyContents
}

// A TargetStateSymlink represents the state of a symlink in the target state.
type TargetStateSymlink struct {
	*LazyLinkname
}

// Apply updates destStateEntry to match t.
func (t *TargetStateAbsent) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	if _, ok := destStateEntry.(*DestStateAbsent); ok {
		return nil
	}
	return mutator.RemoveAll(destStateEntry.Path())
}

// Equal returns true if destStateEntry matches t.
func (t *TargetStateAbsent) Equal(destStateEntry DestStateEntry) bool {
	_, ok := destStateEntry.(*DestStateAbsent)
	return ok
}

// Apply updates destStateEntry to match t. It does not recurse.
func (t *TargetStateDir) Apply(mutator Mutator, destStateEntry DestStateEntry) error {
	if destStateDir, ok := destStateEntry.(*DestStateDir); ok {
		if destStateDir.perm == t.perm {
			return nil
		}
		return mutator.Chmod(destStateDir.Path(), t.perm)
	}
	if err := destStateEntry.Remove(); err != nil {
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