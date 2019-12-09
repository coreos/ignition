# ignition-dracut for Fedora CoreOS

This repo holds custom dracut modules required by Fedora and
RHEL CoreOS for Ignition to work properly.

It's packaged on Fedora together with
[Ignition](https://github.com/coreos/ignition) in the
[ignition](https://src.fedoraproject.org/rpms/ignition)
package.

The easiest way to test it out is to pick up the latest
Fedora CoreOS preview artifact from:

https://getfedora.org/coreos/download/

You can see an example of how to pass a config
on qemu at least in coreos-assembler:

https://github.com/coreos/coreos-assembler/blob/master/src/cmd-run

Note that a lot of things are in flux and subject to rapid
change. E.g. some key names have changed wrt their
equivalents in CoreOS Container Linux.

### Branches

There are two branches:
- `master` works with the `master` branch of Ignition and is
  currently used by Fedora CoreOS, which targets Ignition v2
  (spec 3).
- `spec2x` works with the `spec2x` branch of Ignition and is
  currently used by RHEL CoreOS, which (for now) targets
  Ignition v0.x (spec 2).
