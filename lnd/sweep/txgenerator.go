package sweep

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kaotisk-hund/cjdcoind/blockchain"
	"github.com/kaotisk-hund/cjdcoind/btcutil"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/input"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwallet/chainfee"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinlog/log"
	"github.com/kaotisk-hund/cjdcoind/txscript"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

var (
	// DefaultMaxInputsPerTx specifies the default maximum number of inputs
	// allowed in a single sweep tx. If more need to be swept, multiple txes
	// are created and published.
	DefaultMaxInputsPerTx = 100
)

// txInput is an interface that provides the input data required for tx
// generation.
type txInput interface {
	input.Input
	parameters() Params
}

// inputSet is a set of inputs that can be used as the basis to generate a tx
// on.
type inputSet []input.Input

// generateInputPartitionings goes through all given inputs and constructs sets
// of inputs that can be used to generate a sensible transaction. Each set
// contains up to the configured maximum number of inputs. Negative yield
// inputs are skipped. No input sets with a total value after fees below the
// dust limit are returned.
func generateInputPartitionings(sweepableInputs []txInput,
	relayFeePerKW, feePerKW chainfee.SatPerKWeight,
	maxInputsPerTx int, wallet Wallet) ([]inputSet, er.R) {

	// Sort input by yield. We will start constructing input sets starting
	// with the highest yield inputs. This is to prevent the construction
	// of a set with an output below the dust limit, causing the sweep
	// process to stop, while there are still higher value inputs
	// available. It also allows us to stop evaluating more inputs when the
	// first input in this ordering is encountered with a negative yield.
	//
	// Yield is calculated as the difference between value and added fee
	// for this input. The fee calculation excludes fee components that are
	// common to all inputs, as those wouldn't influence the order. The
	// single component that is differentiating is witness size.
	//
	// For witness size, the upper limit is taken. The actual size depends
	// on the signature length, which is not known yet at this point.
	yields := make(map[wire.OutPoint]int64)
	for _, input := range sweepableInputs {
		size, _, err := input.WitnessType().SizeUpperBound()
		if err != nil {
			return nil, er.Errorf(
				"failed adding input weight: %v", err)
		}

		yields[*input.OutPoint()] = input.SignDesc().Output.Value -
			int64(feePerKW.FeeForWeight(int64(size)))
	}

	sort.Slice(sweepableInputs, func(i, j int) bool {
		// Because of the specific ordering and termination condition
		// that is described above, we place force sweeps at the start
		// of the list. Otherwise we can't be sure that they will be
		// included in an input set.
		if sweepableInputs[i].parameters().Force {
			return true
		}

		return yields[*sweepableInputs[i].OutPoint()] >
			yields[*sweepableInputs[j].OutPoint()]
	})

	// Select blocks of inputs up to the configured maximum number.
	var sets []inputSet
	for len(sweepableInputs) > 0 {
		// Start building a set of positive-yield tx inputs under the
		// condition that the tx will be published with the specified
		// fee rate.
		txInputs := newTxInputSet(
			wallet, feePerKW, relayFeePerKW, maxInputsPerTx,
		)

		// From the set of sweepable inputs, keep adding inputs to the
		// input set until the tx output value no longer goes up or the
		// maximum number of inputs is reached.
		txInputs.addPositiveYieldInputs(sweepableInputs)

		// If there are no positive yield inputs, we can stop here.
		inputCount := len(txInputs.inputs)
		if inputCount == 0 {
			return sets, nil
		}

		// Check the current output value and add wallet utxos if
		// needed to push the output value to the lower limit.
		if err := txInputs.tryAddWalletInputsIfNeeded(); err != nil {
			return nil, err
		}

		// If the output value of this block of inputs does not reach
		// the dust limit, stop sweeping. Because of the sorting,
		// continuing with the remaining inputs will only lead to sets
		// with an even lower output value.
		if !txInputs.enoughInput() {
			log.Debugf("Set value %v (r=%v, c=%v) below dust "+
				"limit of %v", txInputs.totalOutput(),
				txInputs.requiredOutput, txInputs.changeOutput,
				txInputs.dustLimit)
			return sets, nil
		}

		log.Infof("Candidate sweep set of size=%v (+%v wallet inputs), "+
			"has yield=%v, weight=%v",
			inputCount, len(txInputs.inputs)-inputCount,
			txInputs.totalOutput()-txInputs.walletInputTotal,
			txInputs.weightEstimate(true).weight())

		sets = append(sets, txInputs.inputs)
		sweepableInputs = sweepableInputs[inputCount:]
	}

	return sets, nil
}

// createSweepTx builds a signed tx spending the inputs to a the output script.
func createSweepTx(inputs []input.Input, outputPkScript []byte,
	currentBlockHeight uint32, feePerKw chainfee.SatPerKWeight,
	dustLimit btcutil.Amount, signer input.Signer) (*wire.MsgTx, er.R) {

	inputs, estimator := getWeightEstimate(inputs, feePerKw)

	txFee := estimator.fee()

	// Create the sweep transaction that we will be building. We use
	// version 2 as it is required for CSV.
	sweepTx := wire.NewMsgTx(2)

	// Track whether any of the inputs require a certain locktime.
	locktime := int32(-1)

	// We start by adding all inputs that commit to an output. We do this
	// since the input and output index must stay the same for the
	// signatures to be valid.
	var (
		totalInput     btcutil.Amount
		requiredOutput btcutil.Amount
	)
	for _, o := range inputs {
		if o.RequiredTxOut() == nil {
			continue
		}

		sweepTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *o.OutPoint(),
			Sequence:         o.BlocksToMaturity(),
		})
		sweepTx.AddTxOut(o.RequiredTxOut())

		if lt, ok := o.RequiredLockTime(); ok {
			// If another input commits to a different locktime,
			// they cannot be combined in the same transcation.
			if locktime != -1 && locktime != int32(lt) {
				return nil, er.Errorf("incompatible locktime")
			}

			locktime = int32(lt)
		}

		totalInput += btcutil.Amount(o.SignDesc().Output.Value)
		requiredOutput += btcutil.Amount(o.RequiredTxOut().Value)
	}

	// Sum up the value contained in the remaining inputs, and add them to
	// the sweep transaction.
	for _, o := range inputs {
		if o.RequiredTxOut() != nil {
			continue
		}

		sweepTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *o.OutPoint(),
			Sequence:         o.BlocksToMaturity(),
		})

		if lt, ok := o.RequiredLockTime(); ok {
			if locktime != -1 && locktime != int32(lt) {
				return nil, er.Errorf("incompatible locktime")
			}

			locktime = int32(lt)
		}

		totalInput += btcutil.Amount(o.SignDesc().Output.Value)
	}

	// The value remaining after the required output and fees, go to
	// change. Not that this fee is what we would have to pay in case the
	// sweep tx has a change output.
	changeAmt := totalInput - requiredOutput - txFee

	// The txn will sweep the amount after fees to the pkscript generated
	// above.
	if changeAmt >= dustLimit {
		sweepTx.AddTxOut(&wire.TxOut{
			PkScript: outputPkScript,
			Value:    int64(changeAmt),
		})
	}

	// We'll default to using the current block height as locktime, if none
	// of the inputs commits to a different locktime.
	sweepTx.LockTime = currentBlockHeight
	if locktime != -1 {
		sweepTx.LockTime = uint32(locktime)
	}

	// Before signing the transaction, check to ensure that it meets some
	// basic validity requirements.
	//
	// TODO(conner): add more control to sanity checks, allowing us to
	// delay spending "problem" outputs, e.g. possibly batching with other
	// classes if fees are too low.
	btx := btcutil.NewTx(sweepTx)
	if err := blockchain.CheckTransactionSanity(btx); err != nil {
		return nil, err
	}

	hashCache := txscript.NewTxSigHashes(sweepTx)

	// With all the inputs in place, use each output's unique input script
	// function to generate the final witness required for spending.
	addInputScript := func(idx int, tso input.Input) er.R {
		inputScript, err := tso.CraftInputScript(
			signer, sweepTx, hashCache, idx,
		)
		if err != nil {
			return err
		}

		sweepTx.TxIn[idx].Witness = inputScript.Witness

		if len(inputScript.SigScript) != 0 {
			sweepTx.TxIn[idx].SignatureScript = inputScript.SigScript
		}

		return nil
	}

	// Finally we'll attach a valid input script to each csv and cltv input
	// within the sweeping transaction.
	for i, input := range inputs {
		if err := addInputScript(i, input); err != nil {
			return nil, err
		}
	}

	log.Infof("Creating sweep transaction %v for %v inputs (%s) "+
		"using %v sat/kw, tx_weight=%v, tx_fee=%v, parents_count=%v, "+
		"parents_fee=%v, parents_weight=%v",
		sweepTx.TxHash(), len(inputs),
		inputTypeSummary(inputs), int64(feePerKw),
		estimator.weight(), txFee,
		len(estimator.parents), estimator.parentsFee,
		estimator.parentsWeight,
	)

	return sweepTx, nil
}

// getWeightEstimate returns a weight estimate for the given inputs.
// Additionally, it returns counts for the number of csv and cltv inputs.
func getWeightEstimate(inputs []input.Input, feeRate chainfee.SatPerKWeight) (
	[]input.Input, *weightEstimator) {

	// We initialize a weight estimator so we can accurately asses the
	// amount of fees we need to pay for this sweep transaction.
	//
	// TODO(roasbeef): can be more intelligent about buffering outputs to
	// be more efficient on-chain.
	weightEstimate := newWeightEstimator(feeRate)

	// Our sweep transaction will pay to a single segwit p2wkh address,
	// ensure it contributes to our weight estimate. If the inputs we add
	// have required TxOuts, then this will be our change address. Note
	// that if we have required TxOuts, we might end up creating a sweep tx
	// without a change output. It is okay to add the change output to the
	// weight estimate regardless, since the estimated fee will just be
	// subtracted from this already dust output, and trimmed.
	weightEstimate.addP2WKHOutput()

	// For each output, use its witness type to determine the estimate
	// weight of its witness, and add it to the proper set of spendable
	// outputs.
	var sweepInputs []input.Input
	for i := range inputs {
		inp := inputs[i]

		err := weightEstimate.add(inp)
		if err != nil {
			log.Warn(err)

			// Skip inputs for which no weight estimate can be
			// given.
			continue
		}

		// If this input comes with a committed output, add that as
		// well.
		if inp.RequiredTxOut() != nil {
			weightEstimate.addOutput(inp.RequiredTxOut())
		}

		sweepInputs = append(sweepInputs, inp)
	}

	return sweepInputs, weightEstimate
}

// inputSummary returns a string containing a human readable summary about the
// witness types of a list of inputs.
func inputTypeSummary(inputs []input.Input) string {
	// Sort inputs by witness type.
	sortedInputs := make([]input.Input, len(inputs))
	copy(sortedInputs, inputs)
	sort.Slice(sortedInputs, func(i, j int) bool {
		return sortedInputs[i].WitnessType().String() <
			sortedInputs[j].WitnessType().String()
	})

	var parts []string
	for _, i := range sortedInputs {
		part := fmt.Sprintf("%v (%v)",
			*i.OutPoint(), i.WitnessType())

		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}
