Release checklist:

Tagging:
 - [ ] Write release notes in NEWS. Get them reviewed and merged
     - [ ] If doing a branched release, also include a PR to merge the NEWS changes into master
 - [ ] Ensure your local copy is up to date with master and your working directory is clean
 - [ ] Ensure you can sign commits and any yubikeys/smartcards are plugged in
 - [ ] Run `./tag_release.sh <vX.Y.z> <git commit hash>`
 - [ ] Push that tag to GitHub

Packaging:
 - [ ] Create a PR to bump the Ignition spec file in [Fedora](https://src.fedoraproject.org/rpms/ignition).  Bump the `master` branch for 2.x and the `epel7` branch for 0.x.
 - [ ] For 2.x: once the PR merges to master, merge master into the other relevant branches (e.g. f30) then push those.
 - [ ] On each of those branches run `fedpkg build`
 - [ ] Once the builds have finished, submit them to [bodhi](https://bodhi.fedoraproject.org/updates/new), filling in:
   - `ignition` for `Packages`
   - Selecting the build(s) that just completed, except for the rawhide one (which gets submitted automatically)
   - Writing brief release notes like "New upstream release. See release notes at `link to NEWS on GH tag`"
   - Leave `Update name` blank
   - `Type`, `Severity` and `Suggestion` can be left as `unspecified` unless it is a security release. In that case select `security` which the appropriate severity.
   - `Stable karma` and `Unstable` karma can be set to `2` and `-1`, respectively.

GitHub release:
 - [ ] Wait until the Bodhi update shows "Signed :heavy_check_mark:" in the Metadata box.
 - [ ] [File a releng ticket](https://pagure.io/releng/new_issue) based on prior signing tickets for [0.x](https://pagure.io/releng/issue/9185) or [2.x](https://pagure.io/releng/issue/9602).
   - [ ] Update the script and test it locally by dropping the `sigul` lines.
 - [ ] Ping `mboddu` in Freenode `#fedora-coreos`, linking to the ticket
 - [ ] Wait for the ticket to be closed
 - [ ] Download the artifacts and signatures
 - [ ] Verify the signatures
 - [ ] Create a [draft release](https://github.com/coreos/ignition/releases/new) on GitHub and upload all the release artifacts and their signatures. Copy and paste the release notes from NEWS here as well.
 - [ ] Publish the release

Quay release:
  - [ ] Visit the [Quay tags page](https://quay.io/repository/coreos/ignition-validate?tab=tags) and wait for a versioned tag to appear
  - [ ] Click the gear next to the tag, select "Add New Tag", enter `release`, and confirm

Housekeeping:
 - [ ] Vendor the new Ignition version in [mantle](https://github.com/coreos/coreos-assembler/tree/master/mantle)
