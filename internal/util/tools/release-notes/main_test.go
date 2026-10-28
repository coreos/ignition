// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Maintained in https://github.com/coreos/repo-templates
// Do not edit downstream.

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadAndRenderFragments(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "documentation")
	writeFile(t, filepath.Join(dir, "later-fix.bugfix.md"), "Fix the later bug.\n\nThis keeps a second paragraph.")
	writeFile(t, filepath.Join(dir, "second-feature.feature.md"), "Add the second feature")
	writeFile(t, filepath.Join(dir, "first-feature.feature.md"), "<!-- release-note-pr: 10 -->\nAdd the first feature")
	writeFile(t, filepath.Join(dir, "remove-old-api.breaking.md"), "Remove the old API")
	writeFile(t, filepath.Join(dir, "default-behavior.change.md"), "<!-- release-note-pr: 5 -->\nChange the default")

	fragments, err := loadFragments(dir)
	mustNoError(t, err)
	mustLen(t, fragments, 5)
	assert.Equal(t, []string{
		"remove-old-api.breaking.md", "first-feature.feature.md", "second-feature.feature.md", "default-behavior.change.md", "later-fix.bugfix.md",
	}, fragmentNames(fragments))

	expected := `### Breaking changes

- Remove the old API

### Features

- Add the first feature
  ([#10](https://github.com/coreos/ignition/pull/10))
- Add the second feature

### Changes

- Change the default
  ([#5](https://github.com/coreos/ignition/pull/5))

### Bug fixes

- Fix the later bug.

  This keeps a second paragraph.
`
	assert.Equal(t, expected, renderFragments(fragments))
}

func TestRenderFragmentPreservesFencedCode(t *testing.T) {
	note := fragment{
		category: "feature",
		content:  "Add an example:\n\n```go\nfmt.Println(\"hello\")\n```",
		pr:       42,
	}
	expected := "### Features\n\n" +
		"- Add an example:\n\n" +
		"  ```go\n" +
		"  fmt.Println(\"hello\")\n" +
		"  ```\n" +
		"  ([#42](https://github.com/coreos/ignition/pull/42))\n"
	assert.Equal(t, expected, renderFragments([]fragment{note}))
}

func TestLoadFragmentsOrdersSlugs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "zebra.change.md"), "Zebra")
	writeFile(t, filepath.Join(dir, "alpha.change.md"), "Alpha")
	writeFile(t, filepath.Join(dir, "middle.change.md"), "Middle")

	fragments, err := loadFragments(dir)
	mustNoError(t, err)
	assert.Equal(t, []string{"alpha.change.md", "middle.change.md", "zebra.change.md"}, fragmentNames(fragments))
}

func TestLoadFragmentsRejectsMalformedNames(t *testing.T) {
	for _, name := range []string{
		"missing-category.md",
		"UPPERCASE.change.md",
		"under_score.change.md",
		"leading-.change.md",
		"-leading.change.md",
		"double--hyphen.change.md",
		"sequence.1.change.md",
		"valid-slug.docs.md",
		"valid-slug.change.txt",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, name), "A valid note")
			_, err := loadFragments(dir)
			mustError(t, err)
			assert.Contains(t, err.Error(), "invalid fragment name")
		})
	}
}

func TestLoadFragmentsRejectsMalformedContent(t *testing.T) {
	for name, content := range map[string]string{
		"empty":               "",
		"whitespace":          " \n\t",
		"leading blank":       "\nA note",
		"trailing blank":      "A note\n\n",
		"list marker":         "- A note",
		"Windows newlines":    "A note\r\ncontinued",
		"invalid UTF-8":       string([]byte{0xff}),
		"trailing whitespace": "A note ",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "valid-slug.change.md"), content)
			_, err := loadFragments(dir)
			mustError(t, err)
			assert.Contains(t, err.Error(), "invalid fragment")
		})
	}
}

func TestParseContentMetadata(t *testing.T) {
	content, pr, err := parseContent([]byte("<!-- release-note-pr: 2343 -->\nA migrated note\nwith details\n"))
	mustNoError(t, err)
	assert.Equal(t, 2343, pr)
	assert.Equal(t, "A migrated note\nwith details", content)

	content, pr, err = parseContent([]byte("A normal note\n"))
	mustNoError(t, err)
	assert.Zero(t, pr)
	assert.Equal(t, "A normal note", content)
}

func TestParseContentRejectsMalformedMetadata(t *testing.T) {
	for _, content := range []string{
		"<!-- release-note-pr: abc -->\nA note",
		"<!-- release-note-pr: 0 -->\nA note",
		"<!-- release-note-pr: 123-- >\nA note",
		"<!-- release-note-pr: 123 -->",
	} {
		_, _, err := parseContent([]byte(content))
		mustError(t, err)
	}
}

func TestParsePRFromSubject(t *testing.T) {
	for subject, expected := range map[string]int{
		"Merge pull request #123 from coreos/topic": 123,
		"providers: add example support (#456)":     456,
	} {
		t.Run(subject, func(t *testing.T) {
			actual, err := parsePRFromSubject(subject)
			mustNoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
	for _, subject := range []string{
		"providers: add example support",
		"Revert merge pull request #123 from coreos/topic",
		"providers: mention (#123) in the middle",
	} {
		_, err := parsePRFromSubject(subject)
		mustError(t, err)
	}
}

func TestResolvePRFromGit(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(repo, "README.md"), "test repository")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial commit")
	runGit(t, repo, "checkout", "-b", "topic")
	fragmentPath := filepath.Join(repo, "docs", "release-notes.d", "add-example.feature.md")
	mustNoError(t, os.MkdirAll(filepath.Dir(fragmentPath), 0o755))
	writeFile(t, fragmentPath, "Add an example")
	runGit(t, repo, "add", "--", "docs/release-notes.d/add-example.feature.md")
	runGit(t, repo, "commit", "-m", "docs: add example release note")
	runGit(t, repo, "checkout", "main")
	runGit(t, repo, "merge", "--no-ff", "topic", "-m", "Merge pull request #789 from coreos/topic")
	data, err := os.ReadFile(fragmentPath)
	mustNoError(t, err)

	pr, err := resolvePRFromGit(repo, fragmentPath, data, runCommand)
	mustNoError(t, err)
	assert.Equal(t, 789, pr)
}

func TestResolvePRFromGitRejectsWorkingTreeContentMismatch(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	fragmentPath := filepath.Join(repo, "reused-name.change.md")
	writeFile(t, fragmentPath, "Original note")
	runGit(t, repo, "add", "reused-name.change.md")
	runGit(t, repo, "commit", "-m", "change: original note (#123)")
	writeFile(t, fragmentPath, "Uncommitted replacement note")
	data, err := os.ReadFile(fragmentPath)
	mustNoError(t, err)

	_, err = resolvePRFromGit(repo, fragmentPath, data, runCommand)
	mustError(t, err)
	assert.Contains(t, err.Error(), "does not match its exact contents in HEAD")
}

func TestResolvePRFromGitReportsMissingHistory(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(repo, "README.md"), "test repository")
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial commit")
	fragmentPath := filepath.Join(repo, "uncommitted.change.md")
	writeFile(t, fragmentPath, "An uncommitted note")

	data, readErr := os.ReadFile(fragmentPath)
	mustNoError(t, readErr)
	_, err := resolvePRFromGit(repo, fragmentPath, data, runCommand)
	mustError(t, err)
	assert.Contains(t, err.Error(), "must exist in HEAD")
}

func TestResolvePRFromGitSeparatesPathFromOptions(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "--dangerous.change.md")
	var arguments []string
	runner := func(_ string, name string, args ...string) ([]byte, error) {
		assert.Equal(t, "git", name)
		if args[0] == "show" {
			return []byte("safe contents"), nil
		}
		arguments = append([]string(nil), args...)
		return []byte("change: safe lookup (#321)\n"), nil
	}

	pr, err := resolvePRFromGit(repo, path, []byte("safe contents"), runner)
	mustNoError(t, err)
	assert.Equal(t, 321, pr)
	if !assert.Len(t, arguments, 7) {
		return
	}
	assert.Equal(t, "--", arguments[5])
	assert.Equal(t, "--dangerous.change.md", arguments[6])
}

func TestFinalizeReplacesUpcomingAndDeletesFragments(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	writeFile(t, notesPath, `---
nav_order: 9
---

# Release Notes

## Upcoming Ignition 2.29.0 (unreleased)
`+emptyUpcomingBody()+`
## Ignition 2.28.0 (2026-10-08)

### Features

- An older note
`)
	fragmentPath := filepath.Join(fragmentsDir, "azure-tests.change.md")
	writeFile(t, fragmentPath, "<!-- release-note-pr: 2345 -->\nAdd Azure provider unit tests")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)

	mustNoError(t, finalize(notesPath, fragments, "2.29.0", "2026-11-12"))
	data, err := os.ReadFile(notesPath)
	mustNoError(t, err)
	notes := string(data)
	assert.Contains(t, notes, "## Upcoming Ignition 2.30.0 (unreleased)")
	assert.Contains(t, notes, "## Ignition 2.29.0 (2026-11-12)\n\n### Changes\n\n- Add Azure provider unit tests\n  ([#2345](https://github.com/coreos/ignition/pull/2345))")
	assert.Contains(t, notes, "## Ignition 2.28.0 (2026-10-08)")
	_, err = os.Stat(fragmentPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFinalizeMismatchLeavesFilesUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	original := "# Release Notes\n\n## Upcoming Ignition 2.29.0 (unreleased)\n\n## Ignition 2.28.0 (2026-10-08)\n"
	writeFile(t, notesPath, original)
	fragmentPath := filepath.Join(fragmentsDir, "a-note.change.md")
	writeFile(t, fragmentPath, "<!-- release-note-pr: 2345 -->\nA note")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)

	err = finalize(notesPath, fragments, "2.28.0", "2026-11-12")
	mustError(t, err)
	assert.Contains(t, err.Error(), "does not match")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, original, string(data))
	_, statErr := os.Stat(fragmentPath)
	assert.NoError(t, statErr)
}

func TestFinalizeWithoutFragmentsLeavesReleaseNotesUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	original := emptyReleaseNotes("2.29.0")
	writeFile(t, notesPath, original)

	err := finalize(notesPath, nil, "2.29.0", "2026-11-12")
	mustError(t, err)
	assert.Contains(t, err.Error(), "without release-note fragments")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, original, string(data))
}

func TestFinalizeUnresolvedHistoryLeavesInputsUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	originalNotes := emptyReleaseNotes("2.29.0")
	writeFile(t, notesPath, originalNotes)
	fragmentPath := filepath.Join(fragmentsDir, "unmerged-change.change.md")
	fragmentContent := "Describe an unmerged change"
	writeFile(t, fragmentPath, fragmentContent)
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)

	err = finalizeWithOps(notesPath, fragments, "2.29.0", "2026-11-12", filesystemOps{
		remove:         os.Remove,
		beforeMutation: func() error { return nil },
		resolvePR: func(fragment) (int, error) {
			return 0, errors.New("no adding commit found; history may be shallow")
		},
	})
	mustError(t, err)
	assert.Contains(t, err.Error(), "history may be shallow")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, originalNotes, string(data))
	data, readErr = os.ReadFile(fragmentPath)
	mustNoError(t, readErr)
	assert.Equal(t, fragmentContent, string(data))
}

func TestFinalizePopulatedUpcomingLeavesFilesUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	original := `# Release Notes

## Upcoming Ignition 2.29.0 (unreleased)
` + strings.Replace(emptyUpcomingBody(), "### Changes\n\n", "### Changes\n\n- A manually maintained note\n\n", 1) + `
## Ignition 2.28.0 (2026-10-08)
`
	writeFile(t, notesPath, original)
	fragmentPath := filepath.Join(fragmentsDir, "fragment-note.change.md")
	writeFile(t, fragmentPath, "<!-- release-note-pr: 2345 -->\nA fragment note")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)

	err = finalize(notesPath, fragments, "2.29.0", "2026-11-12")
	mustError(t, err)
	assert.Contains(t, err.Error(), "expected empty four-category scaffold")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, original, string(data))
	_, statErr := os.Stat(fragmentPath)
	assert.NoError(t, statErr)
}

func TestFinalizeChangedFragmentLeavesInputsUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	originalNotes := emptyReleaseNotes("2.29.0")
	writeFile(t, notesPath, originalNotes)
	fragmentPath := filepath.Join(fragmentsDir, "original-fragment.change.md")
	writeFile(t, fragmentPath, "<!-- release-note-pr: 2345 -->\nOriginal fragment")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)
	changedFragment := "Changed fragment"
	writeFile(t, fragmentPath, changedFragment)

	err = finalize(notesPath, fragments, "2.29.0", "2026-11-12")
	mustError(t, err)
	assert.Contains(t, err.Error(), "changed before finalization")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, originalNotes, string(data))
	data, readErr = os.ReadFile(fragmentPath)
	mustNoError(t, readErr)
	assert.Equal(t, changedFragment, string(data))
}

func TestFinalizeChangedReleaseNotesLeavesInputsUnchanged(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	writeFile(t, notesPath, emptyReleaseNotes("2.29.0"))
	fragmentPath := filepath.Join(fragmentsDir, "fragment-note.change.md")
	fragmentContent := "<!-- release-note-pr: 2345 -->\nA fragment note"
	writeFile(t, fragmentPath, fragmentContent)
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)
	changedNotes := emptyReleaseNotes("2.29.0") + "\nConcurrent edit\n"

	err = finalizeWithOps(notesPath, fragments, "2.29.0", "2026-11-12", filesystemOps{
		remove: os.Remove,
		beforeMutation: func() error {
			return os.WriteFile(notesPath, []byte(changedNotes), 0o644)
		},
	})
	mustError(t, err)
	assert.Contains(t, err.Error(), "release notes changed before finalization")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, changedNotes, string(data))
	data, readErr = os.ReadFile(fragmentPath)
	mustNoError(t, readErr)
	assert.Equal(t, fragmentContent, string(data))
}

func TestFinalizeDeletionFailureRollsBack(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	originalNotes := emptyReleaseNotes("2.29.0")
	writeFile(t, notesPath, originalNotes)
	firstPath := filepath.Join(fragmentsDir, "first-note.change.md")
	secondPath := filepath.Join(fragmentsDir, "second-note.bugfix.md")
	writeFile(t, firstPath, "<!-- release-note-pr: 1 -->\nFirst note")
	writeFile(t, secondPath, "<!-- release-note-pr: 2 -->\nSecond note")
	mustNoError(t, os.Chmod(firstPath, 0o600))
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)
	removeCalls := 0

	err = finalizeWithOps(notesPath, fragments, "2.29.0", "2026-11-12", filesystemOps{
		remove: func(path string) error {
			removeCalls++
			if removeCalls == 2 {
				return errors.New("injected deletion failure")
			}
			return os.Remove(path)
		},
		beforeMutation: func() error { return nil },
	})
	mustError(t, err)
	assert.Contains(t, err.Error(), "injected deletion failure")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, originalNotes, string(data))
	data, readErr = os.ReadFile(firstPath)
	mustNoError(t, readErr)
	assert.Equal(t, "<!-- release-note-pr: 1 -->\nFirst note", string(data))
	info, statErr := os.Stat(firstPath)
	mustNoError(t, statErr)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	data, readErr = os.ReadFile(secondPath)
	mustNoError(t, readErr)
	assert.Equal(t, "<!-- release-note-pr: 2 -->\nSecond note", string(data))
}

func TestFinalizeRevalidatesEachFragmentBeforeDeletion(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	originalNotes := emptyReleaseNotes("2.29.0")
	writeFile(t, notesPath, originalNotes)
	firstPath := filepath.Join(fragmentsDir, "first-note.change.md")
	secondPath := filepath.Join(fragmentsDir, "second-note.bugfix.md")
	writeFile(t, firstPath, "<!-- release-note-pr: 1 -->\nFirst note")
	writeFile(t, secondPath, "<!-- release-note-pr: 2 -->\nSecond note")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)
	newerSecond := "A newer edit to the second fragment"
	removeCalls := 0

	err = finalizeWithOps(notesPath, fragments, "2.29.0", "2026-11-12", filesystemOps{
		remove: func(path string) error {
			removeCalls++
			if removeCalls == 1 {
				if err := os.Remove(path); err != nil {
					return err
				}
				return os.WriteFile(secondPath, []byte(newerSecond), 0o644)
			}
			return os.Remove(path)
		},
		beforeMutation: func() error { return nil },
	})
	mustError(t, err)
	assert.Contains(t, err.Error(), "changed before deletion")
	data, readErr := os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, originalNotes, string(data))
	data, readErr = os.ReadFile(firstPath)
	mustNoError(t, readErr)
	assert.Equal(t, "<!-- release-note-pr: 1 -->\nFirst note", string(data))
	data, readErr = os.ReadFile(secondPath)
	mustNoError(t, readErr)
	assert.Equal(t, newerSecond, string(data))
}

func TestFinalizeRollbackRefusesToOverwriteNewerEdits(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "release-notes.md")
	fragmentsDir := filepath.Join(dir, "fragments")
	mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
	writeFile(t, notesPath, emptyReleaseNotes("2.29.0"))
	firstPath := filepath.Join(fragmentsDir, "first-note.change.md")
	secondPath := filepath.Join(fragmentsDir, "second-note.bugfix.md")
	writeFile(t, firstPath, "<!-- release-note-pr: 1 -->\nFirst note")
	writeFile(t, secondPath, "<!-- release-note-pr: 2 -->\nSecond note")
	fragments, err := loadFragments(fragmentsDir)
	mustNoError(t, err)
	newerFragment := "A newer fragment created during deletion"
	newerNotes := "A newer release-notes edit during deletion\n"
	removeCalls := 0

	err = finalizeWithOps(notesPath, fragments, "2.29.0", "2026-11-12", filesystemOps{
		remove: func(path string) error {
			removeCalls++
			if removeCalls == 1 {
				return os.Remove(path)
			}
			if err := os.WriteFile(firstPath, []byte(newerFragment), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(notesPath, []byte(newerNotes), 0o644); err != nil {
				return err
			}
			return errors.New("injected deletion failure")
		},
		beforeMutation: func() error { return nil },
	})
	mustError(t, err)
	assert.Contains(t, err.Error(), "restoration conflict: path contains newer")
	assert.Contains(t, err.Error(), "release notes no longer contain the generated contents")
	data, readErr := os.ReadFile(firstPath)
	mustNoError(t, readErr)
	assert.Equal(t, newerFragment, string(data))
	data, readErr = os.ReadFile(notesPath)
	mustNoError(t, readErr)
	assert.Equal(t, newerNotes, string(data))
}

func TestFinalizeRejectsInvalidInputsWithoutDeletingFragments(t *testing.T) {
	for name, input := range map[string][2]string{
		"version": {"2.29", "2026-11-12"},
		"date":    {"2.29.0", "2026-02-30"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			notesPath := filepath.Join(dir, "release-notes.md")
			fragmentsDir := filepath.Join(dir, "fragments")
			mustNoError(t, os.Mkdir(fragmentsDir, 0o755))
			writeFile(t, notesPath, "## Upcoming Ignition 2.29.0 (unreleased)\n\n## Ignition 2.28.0 (2026-10-08)\n")
			fragmentPath := filepath.Join(fragmentsDir, "fix-a-bug.bugfix.md")
			writeFile(t, fragmentPath, "<!-- release-note-pr: 1 -->\nFix a bug")
			fragments, err := loadFragments(fragmentsDir)
			mustNoError(t, err)

			err = finalize(notesPath, fragments, input[0], input[1])
			mustError(t, err)
			_, statErr := os.Stat(fragmentPath)
			assert.NoError(t, statErr)
		})
	}
}

func TestReplaceUpcomingRequiresExactlyOneHeading(t *testing.T) {
	for _, notes := range []string{
		"## Ignition 2.28.0 (2026-10-08)\n",
		"## Upcoming Ignition 2.29.0 (unreleased)\n\n## Upcoming Ignition 2.29.0 (unreleased)\n\n## Ignition 2.28.0 (2026-10-08)\n",
	} {
		_, err := replaceUpcoming(notes, nil, "2.29.0", "2026-11-12")
		mustError(t, err)
	}
}

func TestLoadAndRenderDoNotDeleteFragments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "add-feature.feature.md")
	writeFile(t, path, "Add a feature")
	fragments, err := loadFragments(dir)
	mustNoError(t, err)
	assert.NotEmpty(t, renderFragments(fragments))
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mustNoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, output)
	}
}

func fragmentNames(fragments []fragment) []string {
	names := make([]string, len(fragments))
	for index, fragment := range fragments {
		names[index] = fragment.name
	}
	return names
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}

func mustLen(t *testing.T, value []fragment, length int) {
	t.Helper()
	if len(value) != length {
		t.Fatalf("got length %d, expected %d", len(value), length)
	}
}

func TestPreviewHasNoUnexpectedWhitespace(t *testing.T) {
	note := fragment{category: "change", content: "First line\n  \n  nested Markdown", pr: 1}
	preview := renderFragments([]fragment{note})
	assert.True(t, strings.HasSuffix(preview, "\n"))
	assert.Contains(t, preview, "- First line\n\n    nested Markdown\n  ([#1](https://github.com/coreos/ignition/pull/1))")
	assert.NotContains(t, preview, "\n  \n")
}

func emptyReleaseNotes(version string) string {
	return "# Release Notes\n\n## Upcoming Ignition " + version + " (unreleased)\n" + emptyUpcomingBody() + "\n## Ignition 2.28.0 (2026-10-08)\n"
}
