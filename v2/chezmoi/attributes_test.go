package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirAttributes(t *testing.T) {
	for _, tc := range []struct {
		sourceName string
		da         DirAttributes
	}{
		{
			sourceName: "foo",
			da: DirAttributes{
				Name: "foo",
			},
		},
		{
			sourceName: "dot_foo",
			da: DirAttributes{
				Name: ".foo",
			},
		},
		{
			sourceName: "private_foo",
			da: DirAttributes{
				Name:    "foo",
				Private: true,
			},
		},
		{
			sourceName: "exact_foo",
			da: DirAttributes{
				Name:  "foo",
				Exact: true,
			},
		},
		{
			sourceName: "private_dot_foo",
			da: DirAttributes{
				Name:    ".foo",
				Private: true,
			},
		},
		{
			sourceName: "exact_private_dot_foo",
			da: DirAttributes{
				Name:    ".foo",
				Exact:   true,
				Private: true,
			},
		},
	} {
		t.Run(tc.sourceName, func(t *testing.T) {
			assert.Equal(t, tc.da, ParseDirAttributes(tc.sourceName))
			assert.Equal(t, tc.sourceName, tc.da.SourceName())
		})
	}
}

func TestFileAttributes(t *testing.T) {
	for _, tc := range []struct {
		sourceName string
		fa         FileAttributes
	}{
		{
			sourceName: "foo",
			fa: FileAttributes{
				Name: "foo",
			},
		},
		{
			sourceName: "dot_foo",
			fa: FileAttributes{
				Name: ".foo",
			},
		},
		{
			sourceName: "private_foo",
			fa: FileAttributes{
				Name:    "foo",
				Private: true,
			},
		},
		{
			sourceName: "private_dot_foo",
			fa: FileAttributes{
				Name:    ".foo",
				Private: true,
			},
		},
		{
			sourceName: "empty_foo",
			fa: FileAttributes{
				Name:  "foo",
				Empty: true,
			},
		},
		{
			sourceName: "executable_foo",
			fa: FileAttributes{
				Name:       "foo",
				Executable: true,
			},
		},
		{
			sourceName: "foo.tmpl",
			fa: FileAttributes{
				Name:     "foo",
				Template: true,
			},
		},
		{
			sourceName: "private_executable_dot_foo.tmpl",
			fa: FileAttributes{
				Name:       ".foo",
				Executable: true,
				Private:    true,
				Template:   true,
			},
		},
		{
			sourceName: "run_foo",
			fa: FileAttributes{
				Name: "foo",
				Type: SourceFileTypeScript,
			},
		},
		{
			sourceName: "run_foo.tmpl",
			fa: FileAttributes{
				Name:     "foo",
				Type:     SourceFileTypeScript,
				Template: true,
			},
		},
		{
			sourceName: "run_once_foo",
			fa: FileAttributes{
				Name: "foo",
				Type: SourceFileTypeScript,
				Once: true,
			},
		},
		{
			sourceName: "run_once_foo.tmpl",
			fa: FileAttributes{
				Name:     "foo",
				Type:     SourceFileTypeScript,
				Once:     true,
				Template: true,
			},
		},
		{
			sourceName: "run_dot_foo",
			fa: FileAttributes{
				Name: "dot_foo",
				Type: SourceFileTypeScript,
			},
		},
		{
			sourceName: "symlink_foo",
			fa: FileAttributes{
				Name: "foo",
				Type: SourceFileTypeSymlink,
			},
		},
		{
			sourceName: "symlink_dot_foo",
			fa: FileAttributes{
				Name: ".foo",
				Type: SourceFileTypeSymlink,
			},
		},
		{
			sourceName: "symlink_foo.tmpl",
			fa: FileAttributes{
				Name:     "foo",
				Type:     SourceFileTypeSymlink,
				Template: true,
			},
		},
		{
			sourceName: "encrypted_private_dot_secret_file",
			fa: FileAttributes{
				Name:      ".secret_file",
				Encrypted: true,
				Private:   true,
			},
		},
	} {
		t.Run(tc.sourceName, func(t *testing.T) {
			assert.Equal(t, tc.fa, ParseFileAttributes(tc.sourceName))
			assert.Equal(t, tc.sourceName, tc.fa.SourceName())
		})
	}
}
