// +build windows

package chezmoi

import (
	"os"
)

// WriteFile implements DestDir.WriteFile.
func (d *FSDestDir) WriteFile(name string, data []byte, perm os.FileMode, currData []byte) error {
	return d.FS.WriteFile(name, data, perm)
}
