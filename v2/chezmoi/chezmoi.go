package chezmoi

import (
	"fmt"
	"os"
)

// Suffixes and prefixes.
const (
	dotPrefix        = "dot_"
	emptyPrefix      = "empty_"
	encryptedPrefix  = "encrypted_"
	exactPrefix      = "exact_"
	executablePrefix = "executable_"
	oncePrefix       = "once_"
	privatePrefix    = "private_"
	runPrefix        = "run_"
	symlinkPrefix    = "symlink_"
	TemplateSuffix   = ".tmpl"
)

// Special file names.
const (
	ignoreName       = ".chezmoiignore"
	removeName       = ".chezmoiremove"
	templatesDirName = ".chezmoitemplates"
	versionName      = ".chezmoiversion"

	ignorePrefix = "."
)

const pathSeparator = "/"

// DefaultTemplateOptions are the default template options.
var DefaultTemplateOptions = []string{"missingkey=error"}

// A PersistentState is an interface to a persistent state.
type PersistentState interface {
	Close() error
	Delete(bucket, key []byte) error
	Get(bucket, key []byte) ([]byte, error)
	Set(bucket, key, value []byte) error
}

var modeTypeNames = map[os.FileMode]string{
	0:                 "file",
	os.ModeDir:        "dir",
	os.ModeSymlink:    "symlink",
	os.ModeNamedPipe:  "named pipe",
	os.ModeSocket:     "socket",
	os.ModeDevice:     "device",
	os.ModeCharDevice: "char device",
}

func modeTypeName(mode os.FileMode) string {
	if name, ok := modeTypeNames[mode&os.ModeType]; ok {
		return name
	}
	return fmt.Sprintf("unknown (%d)", mode&os.ModeType)
}
