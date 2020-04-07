package chezmoi

import "os"

// A TargetStateEntry represents the state of an entry in the target state.
type TargetStateEntry interface {
	Apply(Mutator, string, DestStateEntry) error
	Equal(DestStateEntry) (bool, error)
}

// A TargetStateDir represents the state of a directory in the target state.
type TargetStateDir struct {
	perm os.FileMode
}

// A TargetStateFile represents the state of a file in the target state.
type TargetStateFile struct {
	*LazyContents
	perm os.FileMode
}

// A TargetStateSymlink represents the state of a symlink in the target state.
type TargetStateSymlink struct {
	*LazyLinkname
}

// Apply updates destStateEntry to match d. It does not recurse.
func (d *TargetStateDir) Apply(mutator Mutator, targetPath string, destStateEntry DestStateEntry) error {
	if destStateDir, ok := destStateEntry.(*DestStateDir); ok {
		if destStateDir.mode&os.ModePerm == d.perm {
			return nil
		}
		return mutator.Chmod(targetPath, d.perm)
	}
	if destStateEntry != nil {
		if err := mutator.RemoveAll(targetPath); err != nil {
			return err
		}
	}
	return mutator.Mkdir(targetPath, d.perm)
}

// Equal returns true if d is equal to other. It does not recurse.
func (d *TargetStateDir) Equal(other DestStateEntry) (bool, error) {
	destStateDir, ok := other.(*DestStateDir)
	if !ok {
		return false, nil
	}
	return destStateDir.mode&os.ModePerm == d.perm, nil
}
