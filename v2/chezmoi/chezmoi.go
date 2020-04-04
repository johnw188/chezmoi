package chezmoi

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

// A PersistentState is an interface to a persistent state.
type PersistentState interface {
	Close() error
	Delete(bucket, key []byte) error
	Get(bucket, key []byte) ([]byte, error)
	Set(bucket, key, value []byte) error
}
