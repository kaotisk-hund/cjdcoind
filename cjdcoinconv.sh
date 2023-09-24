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
    echo "      cjdwallet           # change the string btcwallet to cjdwallet in the code"
    echo "      rcjdwallet          # change the string cjdwallet back to btcwallet in the code"
    echo "      btcd                # change the string btcd to cjdcoind in the code"
    echo "      rbtcd               # change the string cjdcoind back to btcd in the code"
}

RUN=$SH

imports() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@"github.com/btcsuite/btcd@"github.com/pkt-cash/cjdcoind@g\' $x;
        echo $SED -i -e \'s@"github.com/btcsuite/btcutil@"github.com/pkt-cash/btcutil@g\' $x;
        echo $SED -i -e \'s@"github.com/btcsuite/btcwallet@"github.com/pkt-cash/pktwallet@g\' $x;
    done | $RUN
}
rimports() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@"github.com/pkt-cash/cjdcoind@"github.com/btcsuite/btcd@g\' $x;
        echo $SED -i -e \'s@"github.com/pkt-cash/btcutil@"github.com/btcsuite/btcutil@g\' $x;
        echo $SED -i -e \'s@"github.com/pkt-cash/pktwallet@"github.com/btcsuite/btcwallet@g\' $x;
    done | $RUN
}
btcwallet() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@btcwallet@pktwallet@g\' $x;
    done | $RUN
}
rbtcwallet() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@pktwallet@btcwallet@g\' $x;
    done | $RUN
}
btcd() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@btcd@cjdcoind@g\' $x;
    done | $RUN
}
rbtcd() {
    $FIND ./ -name '*.go' | while read x; do
        echo $SED -i -e \'s@cjdcoind@btcd@g\' $x;
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
    elif test "x$arg" = "xbtcwallet"; then
        btcwallet
        exit 0
    elif test "x$arg" = "xrbtcwallet"; then
        rbtcwallet
        exit 0
    elif test "x$arg" = "xbtcd"; then
        btcd
        exit 0
    elif test "x$arg" = "xrbtcd"; then
        rbtcd
        exit 0
    else
        usage
        die "I don't understand argument $arg"
    fi
done

usage
