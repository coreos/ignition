package xignore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type stateMap map[string]bool

func collectFiles(fsys fs.FS) ([]string, error) {
	files := []string{}

	err := fs.WalkDir(fsys, ".", func(path string, info fs.DirEntry, werr error) error {
		if path == "." {
			return nil
		}
		if werr != nil {
			return werr
		}

		// canonical slash so that patterns match correctly
		path = filepath.FromSlash(path)

		files = append(files, path)
		return nil
	})

	return files, err
}

func (state stateMap) merge(source stateMap) {
	for k, val := range source {
		state[k] = val
	}
}

func (state stateMap) mergeFiles(files []string, value bool) {
	for _, f := range files {
		state[f] = value
	}
}

func (state stateMap) applyPatterns(vfs fs.FS, files []string, patterns []*Pattern) error {
	filesMap := stateMap{}
	dirPatterns := []*Pattern{}
	for _, pattern := range patterns {
		if pattern.IsEmpty() {
			continue
		}
		currFiles := pattern.Matches(files)
		if pattern.IsExclusion() {
			for _, f := range currFiles {
				filesMap[f] = false
			}
		} else {
			for _, f := range currFiles {
				filesMap[f] = true
			}
		}

		// generate dir based patterns
		for _, f := range currFiles {
			// stat with forward slash always - some filesystems don't like
			// backslashes even on Windows
			fi, err := fs.Stat(vfs, filepath.ToSlash(f))
			if err != nil {
				return fmt.Errorf("stat: %w", err)
			}

			if fi.IsDir() {
				strPattern := f + "/**"
				if pattern.IsExclusion() {
					strPattern = "!" + strPattern
				}
				dirPattern := NewPattern(strPattern)
				dirPatterns = append(dirPatterns, dirPattern)
				err := dirPattern.Prepare()
				if err != nil {
					return err
				}
			}
		}
	}

	// handle dirs batch matches
	dirFileMap := stateMap{}
	for _, pattern := range dirPatterns {
		if pattern.IsEmpty() {
			continue
		}
		currFiles := pattern.Matches(files)
		if pattern.IsExclusion() {
			for _, f := range currFiles {
				dirFileMap[f] = false
			}
		} else {
			for _, f := range currFiles {
				dirFileMap[f] = true
			}
		}
	}

	state.merge(dirFileMap)
	state.merge(filesMap)
	return nil
}

func (state stateMap) applyIgnorefile(vfs fs.FS, ignorefile string, nested bool) error {
	// Apply nested ignorefile
	ignorefiles := []string{}

	if nested {
		for file := range state {
			// all subdir ignorefiles
			if strings.HasSuffix(file, ignorefile) {
				ignorefiles = append(ignorefiles, file)
			}
		}
		// Sort by dir deep level
		sort.Slice(ignorefiles, func(i, j int) bool {
			ilen := len(strings.Split(ignorefiles[i], string(os.PathSeparator)))
			jlen := len(strings.Split(ignorefiles[j], string(os.PathSeparator)))
			return ilen < jlen
		})
	} else {
		ignorefiles = []string{ignorefile}
	}

	for _, ifile := range ignorefiles {
		currBasedir := filepath.Dir(ifile)
		currFs := vfs
		if currBasedir != "." {
			var err error
			currFs, err = fs.Sub(vfs, currBasedir)
			if err != nil {
				return err
			}
		}
		patterns, err := loadPatterns(currFs, ignorefile)
		if err != nil {
			return fmt.Errorf("loadPatterns: %w", err)
		}

		currMap := stateMap{}
		currFiles, err := collectFiles(currFs)
		if err != nil {
			return fmt.Errorf("collectFiles: %w", err)
		}
		err = currMap.applyPatterns(currFs, currFiles, patterns)
		if err != nil {
			return fmt.Errorf("applyPatterns: %w", err)
		}

		for nfile, matched := range currMap {
			parentFile := filepath.Join(currBasedir, nfile)
			state[parentFile] = matched
		}
	}

	return nil
}
