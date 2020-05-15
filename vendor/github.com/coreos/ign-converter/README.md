Ignition Spec v1-v2.x.0 <-> v3.0.0 Config Converter
===================================================

## What is this?

This is a tool and library for converting old (spec v1-v2.x.0) Ignition configs
to the new v3.0.0 format, and from v3.0.0 back to v2.2. See [Migrating Configs](https://github.com/coreos/ignition/blob/master/doc/migrating-configs.md)
for details on the changes.

## Extra information when translating from v2 -> v3

Ignition Spec 3 will mount filesystems at the mountpoint specified by path
when running. Filesystems no longer have the name field and files, links,
and directories no longer specify the filesystem by name. This means to
translate filesystems (with the exception of root), you must also provide
a mapping of filesystem name to absolute path, e.g.

`map[string]string{"var": "/var"}`

If you do not, and have a filesystem with the name "var", the translation
will fail.

Conversely, when you translate from spec 3 down to spec 2, we generate names
on the fly based on the path. If no path is specified, it is simply named by
an incrementing integer. This information is not currently being stored,
which means to translate from 3 -> 2 -> 3, you will have to manually provide
the filesystem mapping that we generate.

## TODO

 - Save the generated filesystem mapping, so we can translate seamlessly from
 3 -> 2 -> 3
 - Revisit translated spec versions (currently we translate from v2.3 -> v3.0
 and v3.0 -> v2.2)

## Why is this not part of Ignition?

The old spec versions have bugs that allow specifying configs that don't make
sense. For example, it is valid for a v2.1+ config to specify that a path
should be both a directory and a file. The behavior there is defined by
Ignition's implementation instead of the spec and, in certain edge cases, by the
contents of the filesystem Ignition is operating on.

This means Ignition can't be guaranteed to automatically translate an old
config to an equivalent new config; it can fail at conversion. Since Ignition
internally translates old configs to the latest config, this would mean old
Ignition configs could stop working on newer versions of whatever OS included
Ignition. Additionally, due to the change in how filesystems are handled (new
configs require specifying the path relative to the sysroot that Ignition
should mount the filesystem at), some configs require extra information to
convert from the old versions to the new versions.

This tool exists to allow _mechanical_ translation of old configs to new
configs. If you are also switching operating systems, other changes may be
necessary.

## How can I ensure my old config is translatable?

Most of the problems in old configs stem from specifying illogical things. Make
sure you don't have any duplicate entries. Do not rely on the order in which
files, directories, or links are created. Most configs should be translatable
without problems.
