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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	fragmentsPath    = "docs/release-notes.d"
	releaseNotesPath = "docs/release-notes.md"
	pullRequestURL   = "https://github.com/coreos/ignition/pull/"
)

type category struct {
	name    string
	heading string
}

var categories = []category{
	{name: "breaking", heading: "Breaking changes"},
	{name: "feature", heading: "Features"},
	{name: "change", heading: "Changes"},
	{name: "bugfix", heading: "Bug fixes"},
}

var (
	fragmentNamePattern  = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*)\.(breaking|feature|change|bugfix)\.md$`)
	metadataPattern      = regexp.MustCompile(`^<!-- release-note-pr: ([1-9][0-9]*) -->$`)
	mergeSubjectPattern  = regexp.MustCompile(`^Merge pull request #([1-9][0-9]*)\b`)
	squashSubjectPattern = regexp.MustCompile(`\(#([1-9][0-9]*)\)$`)
	versionPattern       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	upcomingPattern      = regexp.MustCompile(`(?m)^## Upcoming Ignition ([0-9]+\.[0-9]+\.[0-9]+) \(unreleased\)\n`)
)

type fragment struct {
	path     string
	name     string
	slug     string
	category string
	content  string
	data     []byte
	mode     os.FileMode
	pr       int
}

type filesystemOps struct {
	remove         func(string) error
	beforeMutation func() error
	resolvePR      func(fragment) (int, error)
}

type commandRunner func(dir, name string, args ...string) ([]byte, error)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return usageError()
	}

	switch args[0] {
	case "validate":
		if len(args) != 1 {
			return usageError()
		}
		fragments, err := loadFragments(fragmentsPath)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "Validated %d release-note fragment(s).\n", len(fragments))
		return nil
	case "preview":
		if len(args) != 1 {
			return usageError()
		}
		fragments, err := loadFragments(fragmentsPath)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, renderFragments(fragments))
		return err
	case "finalize":
		if len(args) != 3 {
			return usageError()
		}
		fragments, err := loadFragments(fragmentsPath)
		if err != nil {
			return err
		}
		if err := finalize(releaseNotesPath, fragments, args[1], args[2]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "Finalized Ignition %s (%s) with %d release-note fragment(s).\n", args[1], args[2], len(fragments))
		return nil
	default:
		return usageError()
	}
}

func usageError() error {
	return errors.New("usage: release-notes validate | preview | finalize VERSION YYYY-MM-DD")
}

func loadFragments(dir string) ([]fragment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading fragment directory: %w", err)
	}

	var fragments []fragment
	for _, entry := range entries {
		if entry.Name() == "README.md" {
			continue
		}
		if entry.Type() != 0 {
			return nil, fmt.Errorf("invalid entry %q in fragment directory: only regular files are allowed", entry.Name())
		}
		matches := fragmentNamePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("invalid fragment name %q", entry.Name())
		}
		path := filepath.Join(dir, entry.Name())
		data, mode, err := readFileSnapshot(path)
		if err != nil {
			return nil, fmt.Errorf("reading fragment %q: %w", entry.Name(), err)
		}
		if !mode.IsRegular() {
			return nil, fmt.Errorf("invalid entry %q in fragment directory: only regular files are allowed", entry.Name())
		}
		content, pr, err := parseContent(data)
		if err != nil {
			return nil, fmt.Errorf("invalid fragment %q: %w", entry.Name(), err)
		}
		fragments = append(fragments, fragment{
			path: path, name: entry.Name(), slug: matches[1], category: matches[2], content: content, data: data, mode: mode, pr: pr,
		})
	}

	sort.Slice(fragments, func(i, j int) bool {
		left, right := fragments[i], fragments[j]
		if categoryIndex(left.category) != categoryIndex(right.category) {
			return categoryIndex(left.category) < categoryIndex(right.category)
		}
		return left.slug < right.slug
	})
	return fragments, nil
}

func parseContent(data []byte) (string, int, error) {
	if !utf8.Valid(data) {
		return "", 0, errors.New("content is not valid UTF-8")
	}
	content := strings.TrimSuffix(string(data), "\n")
	if strings.Contains(content, "\r") {
		return "", 0, errors.New("content must use Unix line endings")
	}
	pr := 0
	if firstLine, remainder, found := strings.Cut(content, "\n"); found {
		if matches := metadataPattern.FindStringSubmatch(firstLine); matches != nil {
			var err error
			pr, err = strconv.Atoi(matches[1])
			if err != nil {
				return "", 0, errors.New("release-note PR metadata is too large")
			}
			content = remainder
		} else if strings.HasPrefix(firstLine, "<!-- release-note-pr:") {
			return "", 0, errors.New("malformed release-note PR metadata")
		}
	} else if strings.HasPrefix(content, "<!-- release-note-pr:") {
		return "", 0, errors.New("release-note PR metadata must be followed by note content")
	}
	if strings.TrimSpace(content) == "" {
		return "", 0, errors.New("content is empty")
	}
	if strings.TrimSpace(content) != content {
		return "", 0, errors.New("content must not have leading or trailing whitespace or blank lines")
	}
	if strings.HasPrefix(content, "- ") {
		return "", 0, errors.New("content must not include the release-note list marker")
	}
	return content, pr, nil
}

func categoryIndex(name string) int {
	for index, category := range categories {
		if category.name == name {
			return index
		}
	}
	panic("unknown release-note category " + name)
}

func renderFragments(fragments []fragment) string {
	var output strings.Builder
	for _, category := range categories {
		var matching []fragment
		for _, fragment := range fragments {
			if fragment.category == category.name {
				matching = append(matching, fragment)
			}
		}
		if len(matching) == 0 {
			continue
		}
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		fmt.Fprintf(&output, "### %s\n\n", category.heading)
		for _, fragment := range matching {
			output.WriteString(renderFragment(fragment))
			output.WriteString("\n")
		}
	}
	return output.String()
}

func renderFragment(fragment fragment) string {
	lines := strings.Split(fragment.content, "\n")

	var output strings.Builder
	output.WriteString("- ")
	output.WriteString(lines[0])
	for _, line := range lines[1:] {
		output.WriteString("\n")
		if strings.TrimSpace(line) != "" {
			output.WriteString("  ")
			output.WriteString(line)
		}
	}
	if fragment.pr != 0 {
		fmt.Fprintf(&output, "\n  ([#%d](%s%d))", fragment.pr, pullRequestURL, fragment.pr)
	}
	return output.String()
}

func finalize(notesPath string, fragments []fragment, version, date string) error {
	return finalizeWithOps(notesPath, fragments, version, date, filesystemOps{
		remove:         os.Remove,
		beforeMutation: func() error { return nil },
		resolvePR: func(fragment fragment) (int, error) {
			return resolvePRFromGit(".", fragment.path, fragment.data, runCommand)
		},
	})
}

func finalizeWithOps(notesPath string, fragments []fragment, version, date string, ops filesystemOps) error {
	if len(fragments) == 0 {
		return errors.New("cannot finalize a release without release-note fragments")
	}
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version must be in X.Y.Z format, got %q", version)
	}
	if parsed, err := time.Parse("2006-01-02", date); err != nil || parsed.Format("2006-01-02") != date {
		return fmt.Errorf("date must be a valid YYYY-MM-DD date, got %q", date)
	}
	resolvedFragments, err := resolveFragmentPRs(fragments, ops.resolvePR)
	if err != nil {
		return err
	}

	data, originalMode, err := readFileSnapshot(notesPath)
	if err != nil {
		return fmt.Errorf("reading release notes: %w", err)
	}
	updated, err := replaceUpcoming(string(data), resolvedFragments, version, date)
	if err != nil {
		return err
	}

	updatedData := []byte(updated)
	temporaryPath, err := writeTemporaryFile(notesPath, updatedData, originalMode)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := ops.beforeMutation(); err != nil {
		return fmt.Errorf("preparing to update release notes: %w", err)
	}
	if err := revalidateInputs(notesPath, data, originalMode, resolvedFragments); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, notesPath); err != nil {
		return fmt.Errorf("replacing release notes: %w", err)
	}
	var removedFragments []fragment
	for _, fragment := range resolvedFragments {
		if err := revalidateFile(fragment.path, fragment.data, fragment.mode); err != nil {
			changeErr := fmt.Errorf("fragment %q changed before deletion: %w", fragment.name, err)
			return rollbackFinalization(notesPath, data, updatedData, originalMode, removedFragments, changeErr)
		}
		if err := ops.remove(fragment.path); err != nil {
			removeErr := fmt.Errorf("removing consumed fragment %q: %w", fragment.name, err)
			return rollbackFinalization(notesPath, data, updatedData, originalMode, removedFragments, removeErr)
		}
		removedFragments = append(removedFragments, fragment)
	}
	return nil
}

func resolveFragmentPRs(fragments []fragment, resolver func(fragment) (int, error)) ([]fragment, error) {
	resolved := append([]fragment(nil), fragments...)
	for index := range resolved {
		if resolved[index].pr != 0 {
			continue
		}
		if resolver == nil {
			return nil, fmt.Errorf("fragment %q has no PR override and no Git resolver is configured", resolved[index].name)
		}
		pr, err := resolver(resolved[index])
		if err != nil {
			return nil, fmt.Errorf("resolving pull request for fragment %q: %w", resolved[index].name, err)
		}
		if pr <= 0 {
			return nil, fmt.Errorf("resolving pull request for fragment %q: resolver returned invalid PR number %d", resolved[index].name, pr)
		}
		resolved[index].pr = pr
	}
	return resolved, nil
}

func runCommand(dir, name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Dir = dir
	return command.CombinedOutput()
}

func resolvePRFromGit(repoDir, path string, expectedData []byte, run commandRunner) (int, error) {
	repoPath, err := filepath.Abs(repoDir)
	if err != nil {
		return 0, fmt.Errorf("resolving repository path: %w", err)
	}
	fragmentPath := path
	if !filepath.IsAbs(fragmentPath) {
		fragmentPath = filepath.Join(repoPath, fragmentPath)
	}
	fragmentPath, err = filepath.Abs(fragmentPath)
	if err != nil {
		return 0, fmt.Errorf("resolving fragment path: %w", err)
	}
	relativePath, err := filepath.Rel(repoPath, fragmentPath)
	if err != nil {
		return 0, fmt.Errorf("making fragment path repository-relative: %w", err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return 0, fmt.Errorf("fragment path %q is outside repository %q", path, repoDir)
	}
	gitPath := filepath.ToSlash(relativePath)
	headData, err := run(repoPath, "git", "show", "HEAD:./"+gitPath)
	if err != nil {
		return 0, fmt.Errorf("reading HEAD blob for %q: %w: %s; the fragment must exist in HEAD", relativePath, err, strings.TrimSpace(string(headData)))
	}
	if !bytes.Equal(headData, expectedData) {
		return 0, fmt.Errorf("fragment %q does not match its exact contents in HEAD; commit the current fragment before finalizing", relativePath)
	}

	output, err := run(repoPath, "git", "log", "--first-parent", "--diff-filter=A", "-1", "--format=%s", "--", gitPath)
	if err != nil {
		return 0, fmt.Errorf("reading Git history for %q: %w: %s", relativePath, err, strings.TrimSpace(string(output)))
	}
	subject := strings.TrimSpace(string(output))
	if subject == "" {
		return 0, fmt.Errorf("no adding commit found for %q; the fragment may be uncommitted or history may be shallow", relativePath)
	}
	pr, err := parsePRFromSubject(subject)
	if err != nil {
		return 0, fmt.Errorf("adding commit for %q has unparseable subject %q: %w", relativePath, subject, err)
	}
	return pr, nil
}

func parsePRFromSubject(subject string) (int, error) {
	for _, pattern := range []*regexp.Regexp{mergeSubjectPattern, squashSubjectPattern} {
		if matches := pattern.FindStringSubmatch(subject); matches != nil {
			pr, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, errors.New("PR number is too large")
			}
			return pr, nil
		}
	}
	return 0, errors.New("expected a GitHub merge subject or a squash subject ending in (#N)")
}

func readFileSnapshot(path string) ([]byte, os.FileMode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, err
	}
	return data, info.Mode(), nil
}

func writeTemporaryFile(path string, data []byte, mode os.FileMode) (temporaryPath string, retErr error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".release-notes-*.tmp")
	if err != nil {
		return "", fmt.Errorf("creating temporary file for %q: %w", path, err)
	}
	temporaryPath = temporary.Name()
	defer func() {
		if retErr != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return "", fmt.Errorf("setting temporary file permissions for %q: %w", path, err)
	}
	if _, err := temporary.Write(data); err != nil {
		return "", fmt.Errorf("writing temporary file for %q: %w", path, err)
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("syncing temporary file for %q: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("closing temporary file for %q: %w", path, err)
	}
	return temporaryPath, nil
}

func revalidateInputs(notesPath string, notesData []byte, notesMode os.FileMode, fragments []fragment) error {
	if err := revalidateFile(notesPath, notesData, notesMode); err != nil {
		return fmt.Errorf("release notes changed before finalization: %w", err)
	}
	for _, fragment := range fragments {
		if err := revalidateFile(fragment.path, fragment.data, fragment.mode); err != nil {
			return fmt.Errorf("fragment %q changed before finalization: %w", fragment.name, err)
		}
	}
	return nil
}

func revalidateFile(path string, expected []byte, expectedMode os.FileMode) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode() != expectedMode {
		return fmt.Errorf("mode changed from %v to %v", expectedMode, info.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, expected) {
		return errors.New("contents changed")
	}
	return nil
}

func rollbackFinalization(notesPath string, notesData, generatedNotesData []byte, notesMode os.FileMode, removedFragments []fragment, cause error) error {
	errorsSeen := []error{cause}
	for _, fragment := range removedFragments {
		if err := restoreRemovedFragment(fragment); err != nil {
			errorsSeen = append(errorsSeen, fmt.Errorf("restoring fragment %q: %w", fragment.name, err))
		}
	}
	if err := revalidateFile(notesPath, generatedNotesData, notesMode); err != nil {
		errorsSeen = append(errorsSeen, fmt.Errorf("restoring release notes: restoration conflict: release notes no longer contain the generated contents: %w", err))
	} else if err := atomicReplace(notesPath, notesData, notesMode); err != nil {
		errorsSeen = append(errorsSeen, fmt.Errorf("restoring release notes: %w", err))
	}
	return errors.Join(errorsSeen...)
}

func restoreRemovedFragment(fragment fragment) error {
	info, err := os.Lstat(fragment.path)
	if errors.Is(err, os.ErrNotExist) {
		return atomicReplace(fragment.path, fragment.data, fragment.mode)
	}
	if err != nil {
		return err
	}
	if info.Mode() != fragment.mode {
		return fmt.Errorf("restoration conflict: path mode is %v instead of expected %v", info.Mode(), fragment.mode)
	}
	data, err := os.ReadFile(fragment.path)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, fragment.data) {
		return errors.New("restoration conflict: path contains newer or unexpected contents")
	}
	return nil
}

func atomicReplace(path string, data []byte, mode os.FileMode) error {
	temporaryPath, err := writeTemporaryFile(path, data, mode)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replacing %q: %w", path, err)
	}
	return nil
}

func replaceUpcoming(notes string, fragments []fragment, version, date string) (string, error) {
	matches := upcomingPattern.FindAllStringSubmatchIndex(notes, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("expected exactly one Upcoming Ignition heading, found %d", len(matches))
	}
	match := matches[0]
	upcomingVersion := notes[match[2]:match[3]]
	if upcomingVersion != version {
		return "", fmt.Errorf("version %s does not match current Upcoming Ignition %s heading", version, upcomingVersion)
	}
	remainder := notes[match[1]:]
	nextHeading := strings.Index(remainder, "\n## ")
	if nextHeading < 0 {
		return "", errors.New("could not find a release heading after the Upcoming section")
	}
	if upcomingBody := remainder[:nextHeading]; upcomingBody != emptyUpcomingBody() {
		return "", errors.New("current Upcoming section is not the expected empty four-category scaffold")
	}
	suffix := remainder[nextHeading+1:]
	nextVersion, err := nextMinor(version)
	if err != nil {
		return "", err
	}

	var replacement strings.Builder
	fmt.Fprintf(&replacement, "## Upcoming Ignition %s (unreleased)\n", nextVersion)
	replacement.WriteString(emptyUpcomingBody())
	replacement.WriteString("\n")
	fmt.Fprintf(&replacement, "## Ignition %s (%s)", version, date)
	if rendered := renderFragments(fragments); rendered != "" {
		replacement.WriteString("\n\n")
		replacement.WriteString(strings.TrimSuffix(rendered, "\n"))
	}
	replacement.WriteString("\n\n\n")
	replacement.WriteString(suffix)
	return notes[:match[0]] + replacement.String(), nil
}

func emptyUpcomingBody() string {
	var body strings.Builder
	body.WriteString("\n")
	for _, category := range categories {
		fmt.Fprintf(&body, "### %s\n\n", category.heading)
	}
	return body.String()
}

func nextMinor(version string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("version must be in X.Y.Z format, got %q", version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version in %q", version)
	}
	return fmt.Sprintf("%s.%d.0", parts[0], minor+1), nil
}
