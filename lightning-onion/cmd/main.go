package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	"github.com/kaotisk-hund/cjdcoind/chaincfg"
	sphinx "github.com/kaotisk-hund/cjdcoind/lightning-onion"
)

type OnionHopSpec struct {
	Realm     int    `json:"realm"`
	PublicKey string `json:"pubkey"`
	Payload   string `json:"payload"`
}

type OnionSpec struct {
	SessionKey string         `json:"session_key,omitempty"`
	Hops       []OnionHopSpec `json:"hops"`
}

func parseOnionSpec(spec OnionSpec) (*sphinx.PaymentPath, *btcec.PrivateKey, er.R) {
	var path sphinx.PaymentPath
	var binSessionKey []byte
	var err er.R

	if spec.SessionKey != "" {
		binSessionKey, err = util.DecodeHex(spec.SessionKey)
		if err != nil {
			log.Fatalf("Unable to decode the sessionKey %v: %v\n", spec.SessionKey, err)
		}

		if len(binSessionKey) != 32 {
			log.Fatalf("Session key must be a 32 byte hex string: %v\n", spec.SessionKey)
		}
	} else {
		binSessionKey = bytes.Repeat([]byte{'A'}, 32)
	}

	sessionKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), binSessionKey)

	for i, hop := range spec.Hops {
		binKey, err := util.DecodeHex(hop.PublicKey)
		if err != nil || len(binKey) != 33 {
			log.Fatalf("%s is not a valid hex pubkey %s", hop.PublicKey, err)
		}

		pubkey, err := btcec.ParsePubKey(binKey, btcec.S256())
		if err != nil {
			log.Fatalf("%s is not a valid hex pubkey %s", hop.PublicKey, err)
		}

		path[i].NodePub = *pubkey

		payload, err := util.DecodeHex(hop.Payload)
		if err != nil {
			log.Fatalf("%s is not a valid hex payload %s",
				hop.Payload, err)
		}

		hopPayload, err := sphinx.NewHopPayload(nil, payload)
		if err != nil {
			log.Fatalf("unable to make payload: %v", err)
		}

		path[i].HopPayload = hopPayload

		fmt.Fprintf(os.Stderr, "Node %d pubkey %x\n", i, pubkey.SerializeCompressed())
	}
	return &path, sessionKey, nil
}

// main implements a simple command line utility that can be used in order to
// either generate a fresh mix-header or decode and fully process an existing
// one given a private key.
func main() {
	args := os.Args

	assocData := bytes.Repeat([]byte{'B'}, 32)

	if len(args) < 3 {
		fmt.Printf("Usage: %s (generate|decode) <input-file>\n", args[0])
		return
	} else if args[1] == "generate" {
		var spec OnionSpec

		jsonSpec, errr := ioutil.ReadFile(args[2])
		if errr != nil {
			log.Fatalf("Unable to read JSON onion spec from file %v: %v", args[2], errr)
		}

		if errr := json.Unmarshal(jsonSpec, &spec); errr != nil {
			log.Fatalf("Unable to parse JSON onion spec: %v", errr)
		}

		path, sessionKey, err := parseOnionSpec(spec)
		if err != nil {
			log.Fatalf("could not parse onion spec: %v", err)
		}

		msg, err := sphinx.NewOnionPacket(
			path, sessionKey, assocData,
			sphinx.DeterministicPacketFiller,
		)
		if err != nil {
			log.Fatalf("Error creating message: %v", err)
		}

		w := bytes.NewBuffer([]byte{})
		err = msg.Encode(w)
		if err != nil {
			log.Fatalf("Error serializing message: %v", err)
		}

		fmt.Printf("%x\n", w.Bytes())
	} else if args[1] == "decode" {
		binKey, err := util.DecodeHex(args[2])
		if len(binKey) != 32 || err != nil {
			log.Fatalf("Argument not a valid hex private key")
		}

		hexBytes, _ := ioutil.ReadAll(os.Stdin)
		binMsg, err := util.DecodeHex(strings.TrimSpace(string(hexBytes)))
		if err != nil {
			log.Fatalf("Error decoding message: %s", err)
		}

		privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), binKey)
		privKeyECDH := &sphinx.PrivKeyECDH{PrivKey: privkey}
		replayLog := sphinx.NewMemoryReplayLog()
		s := sphinx.NewRouter(
			privKeyECDH, &chaincfg.TestNet3Params, replayLog,
		)

		replayLog.Start()
		defer replayLog.Stop()

		var packet sphinx.OnionPacket
		err = packet.Decode(bytes.NewBuffer(binMsg))

		if err != nil {
			log.Fatalf("Error parsing message: %v", err)
		}
		p, err := s.ProcessOnionPacket(&packet, assocData, 10)
		if err != nil {
			log.Fatalf("Failed to decode message: %s", err)
		}

		w := bytes.NewBuffer([]byte{})
		err = p.NextPacket.Encode(w)

		if err != nil {
			log.Fatalf("Error serializing message: %v", err)
		}
		fmt.Printf("%x\n", w.Bytes())
	}
}
