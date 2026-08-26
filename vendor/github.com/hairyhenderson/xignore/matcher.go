package xignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Matcher xignore matcher
type Matcher struct {
	fs fs.FS
}

// NewMatcher create matcher from custom filesystem
func NewMatcher(fs fs.FS) *Matcher {
	return &Matcher{fs}
}

// NewSystemMatcher create matcher for system filesystem
func NewSystemMatcher() *Matcher {
	return NewMatcher(os.DirFS("."))
}

func dirExists(fsys fs.FS, name string) (bool, error) {
	fi, err := fs.Stat(fsys, name)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err

}

// Matches returns matched files from dir files, basedir not support relative path, eg './foo/bar'.
func (m *Matcher) Matches(basedir string, options *MatchesOptions) (*MatchesResult, error) {
	var err error
	var vfs fs.FS
	if fsys, ok := m.fs.(fs.SubFS); ok {
		vfs, err = fsys.Sub(basedir)
	} else {
		vfs, err = fs.Sub(m.fs, basedir)
	}
	if err != nil {
		return nil, fmt.Errorf("sub: %w", err)
	}

	ignorefile := options.Ignorefile
	if ok, err := dirExists(vfs, "."); !ok || err != nil {
		if err == nil {
			return nil, fmt.Errorf("dirExists %q: %w", basedir, fs.ErrNotExist)
		}
		return nil, fmt.Errorf("dirExists %q: %w", basedir, err)
	}

	// Root filemap
	rootMap := stateMap{}
	files, err := collectFiles(vfs)
	if err != nil {
		return nil, fmt.Errorf("collectFiles: %w", err)
	}
	// Init all files state
	rootMap.mergeFiles(files, false)

	// Apply before patterns
	beforePatterns, err := makePatterns(options.BeforePatterns)
	if err != nil {
		return nil, fmt.Errorf("makePatterns: %w", err)
	}
	err = rootMap.applyPatterns(vfs, files, beforePatterns)
	if err != nil {
		return nil, fmt.Errorf("applyPatterns: %w", err)
	}

	// Apply ignorefile patterns
	err = rootMap.applyIgnorefile(vfs, ignorefile, options.Nested)
	if err != nil {
		return nil, fmt.Errorf("applyIgnorefile: %w", err)
	}

	// Apply after patterns
	afterPatterns, err := makePatterns(options.AfterPatterns)
	if err != nil {
		return nil, fmt.Errorf("makePatterns: %w", err)
	}
	err = rootMap.applyPatterns(vfs, files, afterPatterns)
	if err != nil {
		return nil, fmt.Errorf("applyPatterns: %w", err)
	}

	res, err := makeResult(vfs, basedir, rootMap)
	if err != nil {
		return nil, fmt.Errorf("makeResult: %w", err)
	}
	return res, nil
}

func makeResult(vfs fs.FS, basedir string, fileMap stateMap) (*MatchesResult, error) {
	matchedFiles := []string{}
	unmatchedFiles := []string{}
	matchedDirs := []string{}
	unmatchedDirs := []string{}
	for f, matched := range fileMap {
		if f == "" {
			continue
		}
		fi, err := fs.Stat(vfs, filepath.ToSlash(f))
		if err != nil {
			return nil, fmt.Errorf("stat: %w", err)
		}
		if fi.IsDir() {
			if matched {
				matchedDirs = append(matchedDirs, f)
			} else {
				unmatchedDirs = append(unmatchedDirs, f)
			}
		} else {
			if matched {
				matchedFiles = append(matchedFiles, f)
			} else {
				unmatchedFiles = append(unmatchedFiles, f)
			}
		}
	}

	sort.Strings(matchedFiles)
	sort.Strings(unmatchedFiles)
	sort.Strings(matchedDirs)
	sort.Strings(unmatchedDirs)
	return &MatchesResult{
		BaseDir:        basedir,
		MatchedFiles:   matchedFiles,
		UnmatchedFiles: unmatchedFiles,
		MatchedDirs:    matchedDirs,
		UnmatchedDirs:  unmatchedDirs,
	}, nil
}
