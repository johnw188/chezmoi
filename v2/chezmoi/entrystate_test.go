package chezmoi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs/vfst"
)

func TestEntryStateApplyAndEqual(t *testing.T) {
	for _, tc1 := range []struct {
		name       string
		entryState DestStateEntry
	}{
		{
			name: "dir",
			entryState: &DestStateDir{
				path: "/home/user/foo",
				mode: os.ModeDir | 0755,
			},
		},
		{
			name: "file",
			entryState: &DestStateFile{
				path:     "/home/user/foo",
				mode:     0644,
				contents: []byte("bar"),
			},
		},
		{
			name: "file_empty",
			entryState: &DestStateFile{
				path: "/home/user/foo",
				mode: 0644,
			},
		},
		{
			name: "file_empty_ok",
			entryState: &DestStateFile{
				path:  "/home/user/foo",
				mode:  0644,
				empty: true,
			},
		},
		{
			name: "symlink",
			entryState: &DestDirSymlink{
				path:     "/home/user/foo",
				mode:     0644,
				linkname: "bar",
			},
		},
	} {
		t.Run(tc1.name, func(t *testing.T) {
			for _, tc2 := range []struct {
				name string
				root interface{}
			}{
				{
					name: "not_present",
					root: map[string]interface{}{
						"/home/user": &vfst.Dir{Perm: 0755},
					},
				},
				{
					name: "existing_dir",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Dir{Perm: 0755},
					},
				},
				{
					name: "existing_dir_chmod",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Dir{Perm: 0644},
					},
				},
				{
					name: "existing_file_empty",
					root: map[string]interface{}{
						"/home/user/foo": "",
					},
				},
				{
					name: "existing_file_contents",
					root: map[string]interface{}{
						"/home/user/foo": "baz",
					},
				},
				{
					name: "existing_file_chmod",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.File{
							Perm: 0755,
						},
					},
				},
				{
					name: "existing_symlink",
					root: map[string]interface{}{
						"/home/user/bar": "",
						"/home/user/foo": &vfst.Symlink{Target: "bar"},
					},
				},
				{
					name: "existing_symlink_broken",
					root: map[string]interface{}{
						"/home/user/foo": &vfst.Symlink{Target: "bar"},
					},
				},
			} {
				t.Run(tc2.name, func(t *testing.T) {
					fs, cleanup, err := vfst.NewTestFS(tc2.root)
					require.NoError(t, err)
					defer cleanup()

					// Read the initial entry state from fs.
					initialEntryState, err := NewEntryState(fs, "/home/user/foo")
					require.NoError(t, err)

					// Apply the desired state.
					mutator := NewFSMutator(fs)
					require.NoError(t, tc1.entryState.Apply(mutator, vfst.DefaultUmask, "/home/user/foo", initialEntryState))

					// Verify that the filesystem matches the desired state.
					vfst.RunTests(t, fs, "", entryStateTest(t, tc1.entryState))

					// Read the updated entry state from fs and verify that it is
					// equal to the desired state.
					newEntryState, err := NewEntryState(fs, "/home/user/foo")
					require.NoError(t, err)
					if newEntryState != nil {
						equal1, err := newEntryState.Equal(tc1.entryState)
						require.NoError(t, err)
						require.True(t, equal1)
					}
					equal2, err := tc1.entryState.Equal(newEntryState)
					require.NoError(t, err)
					require.True(t, equal2)
				})
			}
		})
	}
}

func entryStateTest(t *testing.T, e DestStateEntry) vfst.Test {
	switch e := e.(type) {
	case *DestStateDir:
		return vfst.TestPath(e.path,
			vfst.TestIsDir,
			vfst.TestModePerm(e.mode&os.ModePerm),
		)
	case *DestStateFile:
		expectedContents, err := e.Contents()
		require.NoError(t, err)
		if len(expectedContents) == 0 && !e.empty {
			return vfst.TestPath(e.path,
				vfst.TestDoesNotExist,
			)
		}
		return vfst.TestPath(e.path,
			vfst.TestModeIsRegular,
			vfst.TestModePerm(e.mode&os.ModePerm),
			vfst.TestContents(expectedContents),
		)
	case *DestDirSymlink:
		expectedLinkname, err := e.Linkname()
		require.NoError(t, err)
		return vfst.TestPath(e.path,
			vfst.TestModeType(os.ModeSymlink),
			vfst.TestSymlinkTarget(expectedLinkname),
		)
	default:
		return nil
	}
}
