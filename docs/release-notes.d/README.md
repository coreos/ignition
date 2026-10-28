<!-- Maintained in https://github.com/coreos/repo-templates -->
<!-- Do not edit downstream. -->

# Release-note fragments

Every pull request with a user-visible change should add a release-note
fragment to this directory. Name it `<lowercase-kebab-slug>.<category>.md`,
using a short descriptive slug made of lowercase letters, numbers, and single
hyphens. `<category>` is one of:

- `breaking` for breaking changes
- `feature` for new features
- `change` for other behavior or project changes
- `bugfix` for bug fixes

If one pull request needs multiple notes in the same category, give each
fragment a different descriptive slug.

Write the note as the body of a Markdown list item, without the leading `- `.
Multiline Markdown, including fenced code blocks, is supported. The
release-note tool adds the list marker and renders the pull request link on a
separate continuation line after the fragment body.
After the pull request is merged, finalization finds the fragment's adding
commit in Git first-parent history and derives the pull request link from the
merge or squash commit subject. It also verifies that the exact fragment is
present in `HEAD`, so uncommitted fragments or reused filenames cannot inherit
an older pull request. No PR number is needed in a normal fragment.

For exceptional migrations or backports whose adding commit cannot identify
the original pull request, the first line may specify an override such as
`<!-- release-note-pr: 2343 -->`. Do not use this override in the normal
contributor workflow.

Run these commands from the repository root:

```console
go run ./internal/util/tools/release-notes validate
go run ./internal/util/tools/release-notes preview
go run ./internal/util/tools/release-notes finalize 2.29.0 2026-11-12
```

`validate` and `preview` work before merge and do not modify fragments. Preview
omits links that still need Git history resolution and shows links from
explicit overrides. `finalize` requires at least one fragment, resolvable pull
request links, and the version in the current Upcoming heading. It replaces
that section with the rendered release, creates the next minor Upcoming
scaffold, and deletes the consumed fragments. It does not create a Git commit
or use network APIs.
