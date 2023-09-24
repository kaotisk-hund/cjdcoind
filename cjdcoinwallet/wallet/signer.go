// Copyright (c) 2020 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/waddrmgr"
	"github.com/kaotisk-hund/cjdcoind/txscript"
	"github.com/kaotisk-hund/cjdcoind/txscript/params"
	"github.com/kaotisk-hund/cjdcoind/txscript/scriptbuilder"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// PrivKeyTweaker is a function type that can be used to pass in a callback for
// tweaking a private key before it's used to sign an input.
type PrivKeyTweaker func(*btcec.PrivateKey) (*btcec.PrivateKey, er.R)

// ComputeInputScript generates a complete InputScript for the passed
// transaction with the signature as defined within the passed SignDescriptor.
// This method is capable of generating the proper input script for both
// regular p2wkh output and p2wkh outputs nested within a regular p2sh output.
func (w *Wallet) ComputeInputScript(tx *wire.MsgTx, output *wire.TxOut,
	inputIndex int, sigHashes *txscript.TxSigHashes,
	hashType params.SigHashType, tweaker PrivKeyTweaker) (wire.TxWitness,
	[]byte, er.R) {

	// First make sure we can sign for the input by making sure the script
	// in the UTXO belongs to our wallet and we have the private key for it.
	walletAddr, err := w.fetchOutputAddr(output.PkScript)
	if err != nil {
		return nil, nil, err
	}

	pka := walletAddr.(waddrmgr.ManagedPubKeyAddress)
	privKey, err := pka.PrivKey()
	if err != nil {
		return nil, nil, err
	}

	var (
		witnessProgram []byte
		sigScript      []byte
	)

	switch {
	// If we're spending p2wkh output nested within a p2sh output, then
	// we'll need to attach a sigScript in addition to witness data.
	case pka.AddrType() == waddrmgr.NestedWitnessPubKey:
		pubKey := privKey.PubKey()
		pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())

		// Next, we'll generate a valid sigScript that will allow us to
		// spend the p2sh output. The sigScript will contain only a
		// single push of the p2wkh witness program corresponding to
		// the matching public key of this address.
		p2wkhAddr, err := btcutil.NewAddressWitnessPubKeyHash(
			pubKeyHash, w.chainParams,
		)
		if err != nil {
			return nil, nil, err
		}
		witnessProgram, err = txscript.PayToAddrScript(p2wkhAddr)
		if err != nil {
			return nil, nil, err
		}

		bldr := scriptbuilder.NewScriptBuilder()
		bldr.AddData(witnessProgram)
		sigScript, err = bldr.Script()
		if err != nil {
			return nil, nil, err
		}

	// Otherwise, this is a regular p2wkh output, so we include the
	// witness program itself as the subscript to generate the proper
	// sighash digest. As part of the new sighash digest algorithm, the
	// p2wkh witness program will be expanded into a regular p2kh
	// script.
	default:
		witnessProgram = output.PkScript
	}

	// If we need to maybe tweak our private key, do it now.
	if tweaker != nil {
		privKey, err = tweaker(privKey)
		if err != nil {
			return nil, nil, err
		}
	}

	// Generate a valid witness stack for the input.
	witnessScript, err := txscript.WitnessSignature(
		tx, sigHashes, inputIndex, output.Value, witnessProgram,
		hashType, privKey, true,
	)
	if err != nil {
		return nil, nil, err
	}

	return witnessScript, sigScript, nil
}
