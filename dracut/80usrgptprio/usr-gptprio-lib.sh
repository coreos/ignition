# HACK: since we don't have the things that are normally in the root filesystem
# don't check for them.
usable_root() {
  return 0
}
