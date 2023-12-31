#!/bin/sh
# Copyright (c) 2019 Caleb James DeLisle
# Use of this source code is governed by an ISC
# license that can be found in the LICENSE file.

die() { echo "Error " $1; exit 100; }
SED=`which gsed || which sed`
test "x$SED" = "x" && die "sed not found"
SH=`which sh`
test "x$SH" = "x" && die "sh not found"
CAT=`which cat`
test "x$CAT" = "x" && die "cat not found"
FIND=`which find`
test "x$FIND" = "x" && die "find not found"

usage() {
    echo "cjdconv.sh OPTIONS COMMAND"
    echo "  OPTIONS"
    echo "      --dryrun            # Don't actually change anything"
    echo "  COMMANDS"
    echo "      imports             # Update imported files"
    echo "      rimports            # Revert imported files back to btcd"
}

RUN=$SH

imports() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@"github.com/btcsuite/btcd@"github.com/kaotisk-hund/cjdcoind@\' $x;
        echo $SED -i -e \'s@"github.com/btcsuite/btcutil@"github.com/pkt-cash/btcutil@\' $x;
        echo $SED -i -e \'s@"github.com/btcsuite/btcwallet@"github.com/kaotisk-hund/libcjdwallet@\' $x;
        echo $SED -i -e \'s@"github.com/lightninglabs/neutrino@"github.com/pkt-cash/neutrino@\' $x;
    done | $RUN
}
rimports() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@"github.com/kaotisk-hund/cjdcoind@"github.com/btcsuite/btcd@\' $x;
        echo $SED -i -e \'s@"github.com/pkt-cash/btcutil@"github.com/btcsuite/btcutil@\' $x;
        echo $SED -i -e \'s@"github.com/kaotisk-hund/libcjdwallet@"github.com/btcsuite/btcwallet@\' $x;
        echo $SED -i -e \'s@"github.com/pkt-cash/neutrino@"github.com/lightninglabs/neutrino@\' $x;
    done | $RUN
}

for arg in "$@"; do
    if test "x$arg" = "x--dryrun"; then
        RUN=$CAT
    elif test "x$arg" = "ximports"; then
        imports
        exit 0
    elif test "x$arg" = "xrimports"; then
        rimports
        exit 0
    else
        usage
        die "I don't understand argument $arg"
    fi
done

usage
