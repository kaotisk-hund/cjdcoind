#!/bin/bash

#
# This script should be run from the project root
# e.g. ./contrib/deb/build.sh
#
set -e
fpm -n cjdcoind-linux -s dir -t deb -v "$(./bin/pktctl --version | sed 's/.* version //' | tr -d '\n')" ./bin