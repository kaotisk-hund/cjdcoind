package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kaotisk-hund/cjdcoind/blockchain/packetcrypt/difficulty"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinconfig/version"
)

func usage() {
	fmt.Print("Usage: compact2big <target>\n")
}

func main() {
	version.SetUserAgentName("compact2big")
	if len(os.Args) < 2 {
		usage()
		return
	}
	arg := os.Args[1]
	num, err := strconv.ParseUint(arg, 16, 32)
	if err != nil {
		fmt.Printf("Expected hex number, got [%s]\n", arg)
		return
	}
	bigNum := difficulty.CompactToBig(uint32(num))
	fmt.Printf("%s\n", bigNum.Text(16))
}
