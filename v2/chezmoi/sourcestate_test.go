package chezmoi

import (
	"testing"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

func TestSourceStateRead(t *testing.T) {
	for _, tc := range []struct {
		name                string
		root                interface{}
		sourceStateOptions  []SourceStateOption
		expectedSourceState *SourceState
	}{
		{
			name: "empty",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": &vfst.Dir{Perm: 0755},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": &vfst.Dir{Perm: 0755},
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &dirSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/foo",
						attributes: DirAttributes{
							Name: "foo",
						},
					},
				}),
			),
		},
		{
			name: "file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &fileSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/foo",
						attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeFile,
						},
					},
				}),
			),
		},
		{
			name: "script",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"run_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &fileSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/run_foo",
						attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeScript,
						},
					},
				}),
			),
		},
		{
			name: "symlink",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"symlink_foo": "bar",
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &fileSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/symlink_foo",
						attributes: FileAttributes{
							Name: "foo",
							Type: SourceFileTypeSymlink,
						},
					},
				}),
			),
		},
		{
			name: "file_in_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "baz",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &dirSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/foo",
						attributes: DirAttributes{
							Name: "foo",
						},
					},
					"foo/bar": &fileSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/foo/bar",
						attributes: FileAttributes{
							Name: "bar",
							Type: SourceFileTypeFile,
						},
					},
				}),
			),
		},
		{
			name: "chezmoiignore",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					NewPatternSet(
						withAdd("README.md", true),
					),
				),
			),
		},
		{
			name: "chezmoiignore_ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiignore": "README.md\n",
					"README.md":      "",
				},
			},
			expectedSourceState: NewSourceState(
				withIgnore(
					NewPatternSet(
						withAdd("README.md", true),
					),
				),
			),
		},
		{
			name: "chezmoiremove",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiremove": "!*.txt\n",
				},
			},
			expectedSourceState: NewSourceState(
				withRemove(
					NewPatternSet(
						withAdd("*.txt", false),
					),
				),
			),
		},
		{
			name: "chezmoitemplates",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoitemplates": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withTemplates(
					map[string]*template.Template{
						"/home/user/.local/share/chezmoi/.chezmoitemplates/foo": template.Must(template.New("/home/user/.local/share/chezmoi/.chezmoitemplates/foo").Parse("bar")),
					},
				),
			),
		},
		{
			name: "chezmoiversion",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
				},
			},
			expectedSourceState: NewSourceState(
				withMinVersion(
					&semver.Version{
						Major: 1,
						Minor: 2,
						Patch: 3,
					},
				),
			),
		},
		{
			name: "chezmoiversion_multiple",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".chezmoiversion": "1.2.3\n",
					"foo": map[string]interface{}{
						".chezmoiversion": "2.3.4\n",
					},
				},
			},
			expectedSourceState: NewSourceState(
				withEntryStates(map[string]sourceEntryState{
					"foo": &dirSourceState{
						sourcePath: "/home/user/.local/share/chezmoi/foo",
						attributes: DirAttributes{
							Name: "foo",
						},
					},
				}),
				withMinVersion(
					&semver.Version{
						Major: 2,
						Minor: 3,
						Patch: 4,
					},
				),
			),
		},
		{
			name: "ignore_dir",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expectedSourceState: NewSourceState(),
		},
		{
			name: "ignore_file",
			root: map[string]interface{}{
				"/home/user/.local/share/chezmoi": map[string]interface{}{
					".ignore": "",
				},
			},
			expectedSourceState: NewSourceState(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup, err := vfst.NewTestFS(tc.root)
			require.NoError(t, err)
			defer cleanup()

			s := NewSourceState(tc.sourceStateOptions...)
			require.NoError(t, s.Read(fs, "/home/user/.local/share/chezmoi"))
			assert.Equal(t, tc.expectedSourceState, s)
		})
	}
}

func withEntryStates(entryStates map[string]sourceEntryState) SourceStateOption {
	return func(s *SourceState) {
		s.entryStates = entryStates
	}
}

func withIgnore(ignore *PatternSet) SourceStateOption {
	return func(s *SourceState) {
		s.ignore = ignore
	}
}

func withMinVersion(minVersion *semver.Version) SourceStateOption {
	return func(s *SourceState) {
		s.minVersion = minVersion
	}
}

func withRemove(remove *PatternSet) SourceStateOption {
	return func(s *SourceState) {
		s.remove = remove
	}
}

func withTemplates(templates map[string]*template.Template) SourceStateOption {
	return func(s *SourceState) {
		s.templates = templates
	}
}
