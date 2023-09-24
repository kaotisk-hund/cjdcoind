cjdcoind
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

`cjdcoind` is the primary full node *PKT Cash* implementation, written in Go.

The PKT Cash project is currently under active development and considered 
to be beta quality software.

In particular, the development branch of `cjdcoind` is highly experimental, 
and should generally not be used in a production environment or on the
PKT Cash mainnet.

`cjdcoind` is the primary mainnet node software for the PKT blockchain.
It is known to correctly download, validate, and serve the chain,
using rules for block acceptance based on Bitcoin Core, with the
addition of PacketCrypt Proofs. 

It relays newly mined blocks, and individual transactions that have 
not yet made it into a block, as well as maintaining a transaction pool.
All individual transactions admitted to the pool follow rules defined by 
the network operators, which include strict checks to filter transactions
based on miner requirements ("standard" vs "non-standard" transactions).

Unlike other similar software, `cjdcoind` does *NOT* directly include wallet
functionality - this was an intentional design decision.  You will not be
able to make or receive payments with `cjdcoind` directly.

Example wallet functionality is provided in the included, separate,
[pktwallet](https://github.com/pkt-cash/cjdcoind/tree/master/pktwallet) package.

## Requirements

* Google [Go](http://golang.org) (Golang) version 1.14 or higher.
* A somewhat recent release of Git.

## Issue Tracker

* The GitHub [integrated GitHub issue tracker](https://github.com/pkt-cash/cjdcoind/issues) is used for this project.  

## Building

Using `git`, clone the project from the repository:

`git clone https://github.com/pkt-cash/cjdcoind`

Use the `./do` shell script to build `cjdcoind`, `pktwallet`, and `pktctl`.

NOTE: It is highly recommended to use only the toolchain Google distributes
at the [official Go homepage](https://golang.org/dl). Go toolchains provided
by Linux distributions often use different defaults or apply non-standard
patches to the official sources, usually to meet distribution-specific
requirements (for example, Red Hat backports, security fixes, and provides
a different default linker configuration vs. the upstream Google Go package.)

Support can only be provided for binaries compiled from unmodified sources,
using the official (upstream) Google Golang toolchain. We unfortunately are
unable to test and support every distribution specific combination. 

The official Google Golang installer for Linux is always available 
for download [here](https://storage.googleapis.com/golang/getgo/installer_linux).

## Documentation

The documentation for `cjdcoind` is work-in-progress, and available in the [docs](https://github.com/pkt-cash/cjdcoind/tree/master/docs) folder.

## License

`cjdcoind` is licensed under the [Copyfree](http://Copyfree.org) ISC License.
