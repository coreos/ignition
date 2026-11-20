#!/usr/bin/env python3
"""Pick the blackbox subtests a diff can affect and write the test arguments.

Usage: ci/blackbox_select.py --manifest <tsv> --base <ref> --out <args-file>

The kola wrapper (tests/kola/blackbox/test.sh) reads the args file from
KOLA_EXT_DATA. Fail-closed: any path that is not positively classified selects
the whole suite.
"""

import argparse
import os
import re
import subprocess
import sys

FILES = ["positive/files", "negative/files"]
STORAGE = ["positive/partitions", "negative/partitions",
           "positive/filesystems", "negative/filesystems",
           "positive/luks", "negative/luks",
           "positive/regression", "negative/regression"]
FETCH = ["positive/general", "negative/general",
         "positive/proxy", "negative/proxy",
         "positive/timeouts", "negative/timeouts",
         "positive/security", "negative/security"]
CMDLINE = ["positive/cmdline", "negative/cmdline"]
GENERAL = ["positive/general", "negative/general"]

# Go package dir -> blackbox test dirs.  None means the package is not
# exercised by the suite at all.
BUCKETS = {
    # The files stage package also creates users/groups (passwd.go) and
    # systemd units (units.go), so it reaches those fixtures too.
    "internal/exec/stages/files": FILES + ["positive/passwd", "positive/systemd"],
    "internal/exec/stages/disks": STORAGE,
    "internal/partitioners": STORAGE,
    "internal/partitioners/sfdisk": STORAGE,
    "internal/partitioners/sgdisk": STORAGE,
    "internal/exec/stages/mount": ["positive/filesystems",
                                   "negative/filesystems"] + GENERAL,
    "internal/exec/stages/umount": ["positive/systemd", "positive/filesystems",
                                    "negative/filesystems"] + GENERAL,
    # WaitOnDevices is called from the disks stage and GetSystemdVersion from
    # the files stage, so this package reaches storage and files fixtures too.
    "internal/systemd": STORAGE + FILES,
    # internal/distro is shared: it defines the partitioner, mount, mkfs and
    # user-management commands that several stages call.
    "internal/exec/stages/kargs": CMDLINE,
    "internal/providers/cmdline": CMDLINE + GENERAL,
    "internal/providers/system": CMDLINE + GENERAL,
    "internal/exec/stages/fetch": FETCH,
    "internal/exec/stages/fetch_offline": FETCH,
    # Resource fetching also decompresses and hash-verifies file contents,
    # which the files fixtures assert, and decodes data: URLs, which the
    # positive LUKS fixture uses for its key file.
    "internal/resource": FETCH + FILES + ["positive/luks"],
    "internal/providers": GENERAL + ["positive/security",
                                     "negative/security"] + CMDLINE,
    "internal/platform": CMDLINE + GENERAL,
    "validate": None,
    "butane/internal": None,
    "butane/internal/report": None,
    "butane/internal/version": None,
}

PREFIX_BUCKETS = [
    ("internal/providers/", GENERAL + ["positive/security",
                                       "negative/security"] + CMDLINE),
    ("internal/partitioners/", STORAGE),
]

# A change here can move every test's behavior.
SHARED = {
    "config", "butane/config", "butane/base", "butane/translate",
    "internal/log", "internal/util", "internal/exec/util", "internal/state",
    "internal/exec", "internal/exec/stages", "internal/register", "internal/distro",
    "internal/registry", "internal", "internal/apply", "internal/as_user",
    "internal/earlyrand",
}

# Not consumed by the suite.
SKIP_PATH_PREFIXES = ("docs/", ".github/", "ci/", "tests/tmt/", "butane/docs/",
                      "dracut/", "systemd/", "grub2/", "examples/")
SKIP_PACKAGES = {"internal/doc", "internal/util/tools/docs"}
SKIP_SUFFIXES = (".md", ".yml", ".yaml", ".txt", ".fmf")

# Changes that alter how every config version is parsed or upgraded:
# config.Parse always runs the LATEST spec's parser, which walks every older
# translate package, so a config/v3_N change reaches all versions.
VERSION_SENSITIVE = ("config/", "butane/config/", "butane/base/",
                     "butane/translate")

SECONDS_PER_SUBTEST = 3.0  # measured: 4904s / 1641 subtests

_RE2_META = set(".+*?()[]{}^$|\\")


def re2_escape(s):
    """Go's regexp rejects escapes like "\\ " and "\\-"; escape only metachars."""
    return "".join("\\" + c if c in _RE2_META else c for c in s)


def load_manifest(path):
    """dir -> list of (polarity, name, min-version, version)"""
    man = {}
    with open(path) as fh:
        for line in fh:
            parts = line.rstrip("\n").split("\t")
            if len(parts) != 5:
                continue
            d, pol, name, mn, ver = parts
            man.setdefault(d[len("tests/"):] if d.startswith("tests/") else d,
                           []).append((pol, name, mn, ver))
    return man


def git(repo, *args):
    return subprocess.run(["git", "-C", repo] + list(args), check=True,
                          capture_output=True, text=True).stdout


def resolve_base(repo, base):
    for candidate in (base, "origin/" + base):
        try:
            git(repo, "rev-parse", "--verify", "--quiet", candidate)
            return candidate
        except subprocess.CalledProcessError:
            continue
    return None


def classify(path):
    # tests/tmt is not consumed by the suite; check it before the generic
    # tests branch, which would otherwise fall back to the full suite.
    if path.startswith("tests/tmt/"):
        return "skip", None
    parts = path.split("/")
    if parts[0] == "tests":
        # A test package changing selects that bucket; other test changes fail
        # closed.
        if len(parts) >= 3 and parts[1] in ("positive", "negative"):
            return "bucket", ["%s/%s" % (parts[1], parts[2])]
        return "all", None
    if path.startswith(SKIP_PATH_PREFIXES) or path.endswith(SKIP_SUFFIXES):
        return "skip", None
    if path.endswith("_test.go"):
        return "skip", None
    pkg = os.path.dirname(path)
    if pkg in SKIP_PACKAGES:
        return "skip", None
    if pkg in BUCKETS:
        dirs = BUCKETS[pkg]
        return ("skip", None) if dirs is None else ("bucket", dirs)
    for prefix, dirs in PREFIX_BUCKETS:
        if path.startswith(prefix):
            return "bucket", dirs
    return "all", None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--manifest", required=True)
    ap.add_argument("--base", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--repo", default=".")
    args = ap.parse_args()

    man = load_manifest(args.manifest)
    if not man:
        # An empty manifest means generation failed; run everything rather
        # than silently selecting nothing.
        print("manifest is empty; running the full suite")
        open(args.out, "w").close()
        return 0

    base = resolve_base(args.repo, args.base)
    if base is None:
        print("base %r not found; running the full suite" % args.base)
        open(args.out, "w").close()
        return 0

    try:
        files = [f for f in git(args.repo, "diff", "--name-only",
                                "%s...HEAD" % base).splitlines() if f.strip()]
    except subprocess.CalledProcessError:
        # A shallow checkout can hold both tips without their common ancestor.
        print("no merge base for %r; running the full suite" % base)
        open(args.out, "w").close()
        return 0
    mode, dirs = "select", set()
    for f in files:
        kind, bucket = classify(f)
        if kind == "all":
            mode = "all"
        elif kind == "bucket" and bucket:
            dirs.update(bucket)
    if mode == "select" and not dirs:
        # Every changed path was classified as having no blackbox impact
        # (docs, dracut, _test.go). Keep the selection empty so the wrapper
        # runs nothing instead of the full suite.
        mode = "skip"
    if mode == "all":
        dirs = set(man)

    version_sensitive = any(f.startswith(VERSION_SENSITIVE)
                            and not f.endswith("_test.go") for f in files)
    version_mode = "all" if version_sensitive else "min"

    names, subtests = set(), 0
    for d in sorted(dirs):
        for _pol, name, mn, ver in man.get(d, []):
            if version_mode == "min" and mn != ver:
                continue
            # t.Run sanitizes the version separator (a space) to "_".
            names.add(re.sub(r"\s", "_", name.split("/")[0]))
            subtests += 1

    lines = []
    if mode == "all":
        print("selection: full suite (%d subtests)" % subtests)
    elif not names:
        print("selection: nothing to run")
        lines = ["-test.run=^$"]
    else:
        # -test.run matches each "/"-separated element separately, so the first
        # element must select the two top-level tests; anchor the alternatives
        # or "link" also selects "symlink".
        lines.append("-test.run=(TestIgnitionBlackBox|TestIgnitionBlackBoxNegative)/^(%s)$"
                     % "|".join(re2_escape(n) for n in sorted(names)))
        if version_mode == "min":
            lines.append("-test.short")
        print("selection: %d subtests (~%.1f min), dirs: %s"
              % (subtests, subtests * SECONDS_PER_SUBTEST / 60,
                 ", ".join(sorted(dirs))))

    with open(args.out, "w") as fh:
        for line in lines:
            fh.write(line + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
