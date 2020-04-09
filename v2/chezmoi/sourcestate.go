package chezmoi

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
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
	EntryState(vfs.FS, os.FileMode, string) EntryState
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

// Archive writes s to w.
func (s *SourceState) Archive(fs vfs.FS, umask os.FileMode, w *tar.Writer) error {
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

	headerTemplate := tar.Header{
		Uid:        uid,
		Gid:        gid,
		Uname:      Uname,
		Gname:      Gname,
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	}

	for _, targetName := range s.sortedTargetNames() {
		entryState := s.entryStates[targetName].EntryState(fs, umask, targetName)
		if entryState == nil {
			continue
		}
		if err := entryState.Archive(w, &headerTemplate, umask); err != nil {
			return err
		}
	}

	return nil
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
		relPath := strings.TrimPrefix(thisPath, sourcePath+pathSeparator)
		dir, sourceName := path.Split(relPath)
		targetDirName := getTargetDirName(dir)
		switch {
		case info.Name() == ignoreName:
			return s.addPatterns(fs, s.ignore, thisPath, dir)
		case info.Name() == removeName:
			return s.addPatterns(fs, s.remove, thisPath, targetDirName)
		case info.Name() == templatesDirName:
			if err := s.addTemplatesDir(fs, thisPath); err != nil {
				return err
			}
			return filepath.SkipDir
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
		case s.ignore.Match(relPath):
			return nil
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

func (s *SourceState) addPatterns(fs vfs.FS, ps *PatternSet, path, relPath string) error {
	data, err := s.executeTemplate(fs, path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(relPath)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		text := scanner.Text()
		if index := strings.IndexRune(text, '#'); index != -1 {
			text = text[:index]
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		include := true
		if strings.HasPrefix(text, "!") {
			include = false
			text = strings.TrimPrefix(text, "!")
		}
		pattern := filepath.Join(dir, text)
		if err := ps.Add(pattern, include); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func (s *SourceState) addTemplatesDir(fs vfs.FS, path string) error {
	return vfs.Walk(fs, path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			contents, err := fs.ReadFile(path)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(path, path+pathSeparator)
			tmpl, err := template.New(name).Parse(string(contents))
			if err != nil {
				return err
			}
			if s.templates == nil {
				s.templates = make(map[string]*template.Template)
			}
			s.templates[name] = tmpl
			return nil
		case info.IsDir():
			return nil
		default:
			return fmt.Errorf("%s: unsupported file type", path)
		}
	})
}

func (s *SourceState) executeTemplate(fs vfs.FS, path string) ([]byte, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.ExecuteTemplateData(path, data)
}

func (s *SourceState) sortedTargetNames() []string {
	targetNames := make([]string, 0, len(s.entryStates))
	for targetName := range s.entryStates {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	return targetNames
}

// EntryState returns d's entry state.
func (d *dirSourceState) EntryState(fs vfs.FS, umask os.FileMode, path string) EntryState {
	mode := os.ModeDir | 0777
	if d.attributes.Private {
		mode &^= 077
	}
	return &DirState{
		path: path,
		mode: mode &^ umask,
	}
}

// SourcePath returns d's source path.
func (d *dirSourceState) SourcePath() string {
	return d.sourcePath
}

// EntryState returns f's entry state.
func (f *fileSourceState) EntryState(fs vfs.FS, umask os.FileMode, path string) EntryState {
	switch f.attributes.Type {
	case SourceFileTypeFile:
		mode := os.FileMode(0666)
		if f.attributes.Executable {
			mode |= 0111
		}
		if f.attributes.Private {
			mode &^= 077
		}
		// FIXME templates
		// FIXME encrypted
		return &FileState{
			path: path,
			mode: mode,
			contentsFunc: func() ([]byte, error) {
				return fs.ReadFile(f.sourcePath)
			},
		}
	case SourceFileTypeSymlink:
		return &SymlinkState{
			path: path,
			mode: os.ModeSymlink,
			linknameFunc: func() (string, error) {
				linknameBytes, err := fs.ReadFile(f.sourcePath)
				if err != nil {
					return "", err
				}
				return string(linknameBytes), nil
			},
		}
	default:
		return nil
	}
}

// SourcePath returns f's source path.
func (f *fileSourceState) SourcePath() string {
	return f.sourcePath
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
