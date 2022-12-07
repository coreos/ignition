#!/usr/bin/env python3

'''
    Tiny dumb script that generates virtual bundled `Provides` from a repo that
    uses go modules and vendoring.
'''

import sys
import re


def main():
    repos = get_repos_from_go_mod()
    print_provides_from_modules_txt(repos)


def get_repos_from_go_mod():
    repos = {}
    in_reqs = False
    for line in open('go.mod'):
        line = line.strip()
        if in_reqs and line.startswith(')'):
            break
        if not in_reqs:
            if line.startswith('require ('):
                in_reqs = True
            continue
        req = line.split()

        repo = req[0]
        tag = req[1]

        repos[repo] = go_mod_tag_to_rpm_provides_version(tag)

    return repos


def go_mod_tag_to_rpm_provides_version(tag):

    # go.mod tags are either exact git tags, or may be "pseudo-versions". We
    # want to convert these tags to something resembling a version string that
    # RPM won't fail on. For more information, see
    # https://golang.org/cmd/go/#hdr-Pseudo_versions and following sections.

    # trim off any +incompatible
    if tag.endswith('+incompatible'):
        tag = tag[:-len('+incompatible')]

    # git tags are normally of the form v$VERSION
    if tag.startswith('v'):
        tag = tag[1:]

    # is this a pseudo-version? e.g. v0.0.0-20181031085051-9002847aa142
    m = re.match("(.*)-([0-9.]+)-([a-f0-9]{12})", tag)
    if m:
        # rpm doesn't like multiple dashes in the version, so just merge the
        # timestamp and the commit checksum into the "release" field
        tag = f"{m.group(1)}-{m.group(2)}.git{m.group(3)}"

    return tag


def print_provides_from_modules_txt(repos):

    for line in open('vendor/modules.txt'):
        if line.startswith('#'):
            continue
        gopkg = line.strip()
        repo = lookup_repo_for_pkg(repos, gopkg)
        if not repo:
            # must be a pkg for tests only; ignore
            continue
        tag = repos[repo]
        print(f"Provides: bundled(golang({gopkg})) = {tag}")


def lookup_repo_for_pkg(repos, gopkg):
    for repo in repos:
        if gopkg.startswith(repo):
            return repo


if __name__ == '__main__':
    sys.exit(main())
