#!/bin/bash
set -euxo pipefail

# Send out a warning message to contact the distribution and exit.
# Distributions implementing Ignition kargs can override this binary
# with the actual implementation.
echo -e "Ignition kernelArguments is not supported by this Linux distribution.\nAsk your distribution to implement ignition-kargs-helper."
exit 1
