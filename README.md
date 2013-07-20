# bootengine for CoreOS

This repo holds all of the code needed to create and test the initrd boot
system for CoreOS that handles "reboot to revert".

## Usage

The bootloader will pass the magic cmdline "root=gptprio:" and then the initrd
will figure out which root filesystem to use.

## Running tests

```
./test
```
