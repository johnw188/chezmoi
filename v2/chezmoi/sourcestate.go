package chezmoi

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/coreos/go-semver/semver"
	vfs "github.com/twpayne/go-vfs"
)

// DefaultTemplateOptions are the default template options.
var DefaultTemplateOptions = []string{"missingkey=error"}

type sourceEntryState interface {
	SourcePath() string
}

type dirSourceState struct {
	sourcePath string
	attributes DirAttributes
}

type fileSourceState struct {
	sourcePath string
	attributes FileAttributes
}

// A SourceState is a source state.
type SourceState struct {
	entryStates     map[string]sourceEntryState
	gpg             *GPG
	ignore          *PatternSet
	minVersion      *semver.Version
	remove          *PatternSet
	templateData    interface{}
	templateFuncs   template.FuncMap
	templateOptions []string
	templates       map[string]*template.Template
}

// A SourceStateOption sets an option on a source state.
type SourceStateOption func(*SourceState)

// WithTemplateData sets the template data.
func WithTemplateData(templateData interface{}) SourceStateOption {
	return func(s *SourceState) {
		s.templateData = templateData
	}
}

// WithTemplateFuncs sets the template functions.
func WithTemplateFuncs(templateFuncs template.FuncMap) SourceStateOption {
	return func(s *SourceState) {
		s.templateFuncs = templateFuncs
	}
}

// WithTemplateOptions sets the template options.
func WithTemplateOptions(templateOptions []string) SourceStateOption {
	return func(s *SourceState) {
		s.templateOptions = templateOptions
	}
}

// WithTemplates sets the templates.
func WithTemplates(templates map[string]*template.Template) SourceStateOption {
	return func(s *SourceState) {
		s.templates = templates
	}
}

// NewSourceState creates a new source state with the given options.
func NewSourceState(options ...SourceStateOption) *SourceState {
	s := &SourceState{
		entryStates:     make(map[string]sourceEntryState),
		ignore:          NewPatternSet(),
		remove:          NewPatternSet(),
		templateOptions: DefaultTemplateOptions,
	}
	for _, o := range options {
		o(s)
	}
	return s
}

// ExecuteTemplateData returns the result of executing template data.
func (s *SourceState) ExecuteTemplateData(name string, data []byte) ([]byte, error) {
	tmpl, err := template.New(name).Option(s.templateOptions...).Funcs(s.templateFuncs).Parse(string(data))
	if err != nil {
		return nil, err
	}
	for name, t := range s.templates {
		tmpl, err = tmpl.AddParseTree(name, t.Tree)
		if err != nil {
			return nil, err
		}
	}
	output := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(output, name, s.templateData); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// Read reads a source state from sourcePath in fs.
func (s *SourceState) Read(fs vfs.FS, sourcePath string) error {
	return vfs.Walk(fs, sourcePath, func(thisPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if thisPath == sourcePath {
			return nil
		}
		switch {
		case info.Name() == ignoreName:
			// FIXME
			return nil
		case info.Name() == removeName:
			// FIXME
			return nil
		case info.Name() == templatesDirName:
			// FIXME
			return nil
		case info.Name() == versionName:
			data, err := fs.ReadFile(thisPath)
			if err != nil {
				return err
			}
			version, err := semver.NewVersion(strings.TrimSpace(string(data)))
			if err != nil {
				return err
			}
			if s.minVersion == nil || s.minVersion.LessThan(*version) {
				s.minVersion = version
			}
			return nil
		case strings.HasPrefix(info.Name(), ignorePrefix):
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath := strings.TrimPrefix(thisPath, sourcePath+pathSeparator)
		dir, sourceName := path.Split(relPath)
		targetDirName := getTargetDirName(dir)
		switch {
		case info.IsDir():
			dirAttributes := ParseDirAttributes(sourceName)
			targetPath := path.Join(targetDirName, dirAttributes.Name)
			if ses, ok := s.entryStates[targetPath]; ok {
				return fmt.Errorf("%s: duplicated in source state: %s and %s", targetPath, ses.SourcePath(), sourcePath)
			}
			s.entryStates[targetPath] = &dirSourceState{
				sourcePath: thisPath,
				attributes: dirAttributes,
			}
			return nil
		case info.Mode().IsRegular():
			fileAttributes := ParseFileAttributes(sourceName)
			targetPath := path.Join(targetDirName, fileAttributes.Name)
			if ses, ok := s.entryStates[targetPath]; ok {
				return fmt.Errorf("%s: duplicated in source state: %s and %s", targetPath, ses.SourcePath(), sourcePath)
			}
			s.entryStates[targetPath] = &fileSourceState{
				sourcePath: thisPath,
				attributes: fileAttributes,
			}
			return nil
		default:
			return fmt.Errorf("%s: unsupported file type", thisPath)
		}
	})
}

// SourcePath returns d's source path.
func (d *dirSourceState) SourcePath() string {
	return d.sourcePath
}

// SourcePath returns f's source path.
func (f *fileSourceState) SourcePath() string {
	return f.sourcePath
}

func getTarHeaderTemplate() *tar.Header {
	var (
		now   = time.Now()
		uid   int
		gid   int
		Uname string
		Gname string
	)

	// Attempt to lookup the current user. Ignore errors because the defaults
	// are reasonable.
	if currentUser, err := user.Current(); err == nil {
		uid, _ = strconv.Atoi(currentUser.Uid)
		gid, _ = strconv.Atoi(currentUser.Gid)
		Uname = currentUser.Username
		if group, err := user.LookupGroupId(currentUser.Gid); err != nil {
			Gname = group.Name
		}
	}

	return &tar.Header{
		Uid:        uid,
		Gid:        gid,
		Uname:      Uname,
		Gname:      Gname,
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	}
}

func getTargetDirName(dir string) string {
	sourceNames := strings.Split(dir, pathSeparator)
	targetNames := make([]string, 0, len(sourceNames))
	for _, sourceName := range sourceNames {
		dirAttributes := ParseDirAttributes(sourceName)
		targetNames = append(targetNames, dirAttributes.Name)
	}
	return strings.Join(targetNames, pathSeparator)
}
