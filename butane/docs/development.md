---
nav_order: 10
---

# Developing Butane
{: .no_toc }

1. TOC
{:toc}

## Project layout

Internally, Butane has a versioned `base` component which contains support for
a particular Ignition spec version, plus distro-independent sugar. New base
functionality is added only to the experimental base package. Eventually the
experimental base package is stabilized and a new experimental package
created. The base component is versioned independently of any particular
distro, and its versions are not exposed to the user. Client code should
not need to import anything from `base`.

Each config variant/version pair corresponds to a `config` package, which
derives either from a `base` package or from another `config` package. New
functionality is similarly added only to an experimental config version,
which is eventually stabilized and a new experimental version created.
(This will often happen when the underlying package is stabilized.) A
`config` package can contain sugar or validation logic specific to a distro
(for example, additional properties for configuring etcd).

Packages outside the Butane repository can implement additional config versions
by deriving from a `base` or `config` package and registering their
variant/version pair with `config`.

- `config/` &mdash;
  Top-level `TranslateBytes()` function that determines which config version
  to parse and emit. Clients should typically use this to translate configs.

- `config/common/` &mdash;
  Common definitions for all spec versions, including translate options
  structs and error definitions.

- `config/*/vX_Y/` &mdash;
  User facing definitions of the spec. Each is derived from another config
  package or from a base package. Each one defines its own translate
  functions to be registered in the `config` package. Clients can use
  these directly if they want to translate a specific spec version.

- `config/util/` &mdash;
  Utility code for implementing config packages, including the
  (un)marshaling helpers. Clients don't need to import this unless they're
  implementing an out-of-tree config version.

- `base/` &mdash;
  Distro-agnostic code targeting individual Ignition spec versions. Clients
  don't need to import this unless they're implementing an out-of-tree
  config version.

- `internal/` &mdash;
  `main`, non-exported code.

## Adding sugar

Sugar implementations should generally translate the sugar into a fresh Ignition config struct, then use Ignition config merging to merge that struct with the user's config.  The desugared struct should be the merge parent and the user's config the child, allowing the user to override field values produced by desugaring.

This approach may not always be suitable, since Ignition's config merging isn't always expressive enough.  In that case, it may be necessary to directly modify the user's Ignition config struct.

## Creating a release

Create a [release checklist](https://github.com/coreos/butane/issues/new?template=release-checklist.md) and follow those steps.

## The build process

Note that the binaries released in this repository are not built using the `build` script from this repository
but using a `butane.spec` maintained in [Fedora rpms/butane](https://src.fedoraproject.org/rpms/butane).
This build process uses the [go-rpm-macros](https://pagure.io/go-rpm-macros) to set up the Go build environment and is
subject to the [Golang Packaging Guidelines](https://docs.fedoraproject.org/en-US/packaging-guidelines/Golang/).

Consult the [Package Maintenance Guide](https://docs.fedoraproject.org/en-US/package-maintainers/Package_Maintenance_Guide/)
and the [Pull Requests Guide](https://docs.fedoraproject.org/en-US/ci/pull-requests/) if you want to contribute to the build process.

In case you have trouble with the aforementioned standard Pull Request Guide, consult the Pagure documentation on the 
[Remote Git to Pagure pull request](https://docs.pagure.org/pagure/usage/pull_requests.html#remote-git-to-pagure-pull-request) workflow.

## Bumping spec versions

Create a new [stabilization checklist](https://github.com/coreos/butane/issues/new?template=stabilize-checklist.md) and follow the steps there.
