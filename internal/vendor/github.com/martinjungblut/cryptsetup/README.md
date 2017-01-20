# Go bindings for libcryptsetup

## Rationale
A number of projects have been using go's os/exec package  to interface with the [cryptsetup](https://gitlab.com/cryptsetup/cryptsetup "cryptsetup upstream repository") tools.
This creates additional dependencies, as the programme now depends on the cryptsetup tools being around and installed to a valid path.
Also, it uses subprocessing, so performance is hurt, and it's sometimes hard to control cryptsetup's finer grained options through the command line interface it exposes.

This project is an attempt to create a Go interface for libcryptsetup, providing a clean and object-oriented environment that is both practical, correct and easy to work with.


## What's planned
The following function calls will be implemented for the first release (1.0):

 1. crypt_init()
 2. crypt_format() with plain, LUKS, and Loop-AES types supported
 3. crypt_load()
 4. crypt_activate_by_passphrase()
 5. crypt_activate_by_keyfile_offset()
 6. crypt_activate_by_keyfile()
 7. crypt_activate_by_volume_key()
 8. crypt_deactivate()


## What's been done already

 1. crypt_init()
 2. crypt_format() with LUKS support