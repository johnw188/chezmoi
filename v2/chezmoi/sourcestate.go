package chezmoi

// FIXME accumulate all source state warnings/errors

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	vfs "github.com/twpayne/go-vfs"
)

// A SourceState is a source state.
type SourceState struct {
	entries map[string]SourceStateEntry
	// gpg             *GPG // FIXME
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
		entries:         make(map[string]SourceStateEntry),
		ignore:          NewPatternSet(),
		remove:          NewPatternSet(),
		templateOptions: DefaultTemplateOptions,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ApplyAll updates targetDir in fs to match s using destDir. FIXME comment
func (s *SourceState) ApplyAll(destDir DestDir, umask os.FileMode, targetDir string) error {
	for _, targetName := range s.sortedTargetNames() {
		if err := s.ApplyOne(destDir, umask, targetDir, targetName); err != nil {
			return err
		}
	}
	return nil
}

// ApplyOne updates targetName in targetDir on fs to match s using destDir. FIXME comment
func (s *SourceState) ApplyOne(destDir DestDir, umask os.FileMode, targetDir, targetName string) error {
	targetPath := path.Join(targetDir, targetName)
	destStateEntry, err := NewDestStateEntry(destDir, targetPath)
	if err != nil {
		return err
	}
	targetStateEntry := s.entries[targetName].TargetStateEntry(umask)
	if err != nil {
		return err
	}
	if err := targetStateEntry.Apply(destDir, destStateEntry); err != nil {
		return err
	}
	if targetStateDir, ok := targetStateEntry.(*TargetStateDir); ok {
		if targetStateDir.exact {
			infos, err := destDir.ReadDir(targetPath)
			if err != nil {
				return err
			}
			baseNames := make([]string, 0, len(infos))
			for _, info := range infos {
				if baseName := info.Name(); baseName != "." && baseName != ".." {
					baseNames = append(baseNames, baseName)
				}
			}
			sort.Strings(baseNames)
			for _, baseName := range baseNames {
				if _, ok := s.entries[path.Join(targetName, baseName)]; !ok {
					if err := destDir.RemoveAll(path.Join(targetPath, baseName)); err != nil {
						return err
					}
				}
			}
		}
	}
	// FIXME chezmoiremove
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
func (s *SourceState) Read(fs vfs.FS, sourceDir string) error {
	sourceDirPrefix := filepath.ToSlash(sourceDir) + pathSeparator
	return vfs.Walk(fs, sourceDir, func(sourcePath string, info os.FileInfo, err error) error {
		sourcePath = filepath.ToSlash(sourcePath)
		if err != nil {
			return err
		}
		if sourcePath == sourceDir {
			return nil
		}
		relPath := strings.TrimPrefix(sourcePath, sourceDirPrefix)
		dir, sourceName := path.Split(relPath)
		targetDirName := getTargetDirName(dir)
		switch {
		case info.Name() == ignoreName:
			return s.addPatterns(fs, s.ignore, sourcePath, dir)
		case info.Name() == removeName:
			return s.addPatterns(fs, s.remove, sourcePath, targetDirName)
		case info.Name() == templatesDirName:
			if err := s.addTemplatesDir(fs, sourcePath); err != nil {
				return err
			}
			return filepath.SkipDir
		case info.Name() == versionName:
			data, err := fs.ReadFile(sourcePath)
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
		case info.IsDir():
			dirAttributes := ParseDirAttributes(sourceName)
			targetName := path.Join(targetDirName, dirAttributes.Name)
			if s.ignore.Match(targetName) {
				return nil
			}
			if sourceStateEntry, ok := s.entries[targetName]; ok {
				return &duplicateTargetError{
					targetName: targetName,
					sourcePaths: []string{
						sourceStateEntry.Path(),
						sourcePath,
					},
				}
			}
			s.entries[targetName] = &SourceStateDir{
				path:       sourcePath,
				attributes: dirAttributes,
			}
			return nil
		case info.Mode().IsRegular():
			fileAttributes := ParseFileAttributes(sourceName)
			targetName := path.Join(targetDirName, fileAttributes.Name)
			if s.ignore.Match(targetName) {
				return nil
			}
			if sourceStateEntry, ok := s.entries[targetName]; ok {
				return &duplicateTargetError{
					targetName: targetName,
					sourcePaths: []string{
						sourceStateEntry.Path(),
						sourcePath,
					},
				}
			}
			s.entries[targetName] = &SourceStateFile{
				fs:         fs,
				path:       sourcePath,
				attributes: fileAttributes,
				lazyContents: &lazyContents{
					contentsFunc: func() ([]byte, error) {
						return fs.ReadFile(sourcePath)
					},
				},
			}
			return nil
		default:
			return &unsupportedFileTypeError{
				path: sourcePath,
				mode: info.Mode(),
			}
		}
	})
}

// Remove removes everything in targetDir that matches s's remove pattern set.
func (s *SourceState) Remove(fs vfs.FS, destDir DestDir, umask os.FileMode, targetDir string) error {
	// Build a set of targets to remove.
	targetDirPrefix := targetDir + pathSeparator
	targetPathsToRemove := NewStringSet()
	for include := range s.remove.includes {
		matches, err := fs.Glob(path.Join(targetDir, include))
		if err != nil {
			return err
		}
		for _, match := range matches {
			// Don't remove targets that are excluded from remove.
			if !s.remove.Match(strings.TrimPrefix(match, targetDirPrefix)) {
				continue
			}
			targetPathsToRemove.Add(match)
		}
	}

	sortedTargetPathsToRemove := targetPathsToRemove.Elements()
	sort.Strings(sortedTargetPathsToRemove)
	for _, targetPath := range sortedTargetPathsToRemove {
		if err := destDir.RemoveAll(targetPath); err != nil {
			return err
		}
	}

	return nil
}

// Evaluate evaluates every target state entry in s.
func (s *SourceState) Evaluate(umask os.FileMode) error {
	for _, targetName := range s.sortedTargetNames() {
		if err := s.entries[targetName].TargetStateEntry(umask).Evaluate(); err != nil {
			return err
		}
	}
	return nil
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

func (s *SourceState) addTemplatesDir(fs vfs.FS, templateDir string) error {
	templateDirPrefix := filepath.ToSlash(templateDir) + pathSeparator
	return vfs.Walk(fs, templateDir, func(templatePath string, info os.FileInfo, err error) error {
		templatePath = filepath.ToSlash(templatePath)
		if err != nil {
			return err
		}
		switch {
		case info.Mode().IsRegular():
			contents, err := fs.ReadFile(templatePath)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(templatePath, templateDirPrefix)
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
			return &unsupportedFileTypeError{
				path: templatePath,
				mode: info.Mode(),
			}
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
	targetNames := make([]string, 0, len(s.entries))
	for targetName := range s.entries {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	return targetNames
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
