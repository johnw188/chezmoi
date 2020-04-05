package chezmoi

var (
	_ Mutator = &AnyMutator{}
	_ Mutator = &DebugMutator{}
	_ Mutator = &FSMutator{}
	_ Mutator = &NullMutator{}
	_ Mutator = &VerboseMutator{}
)
