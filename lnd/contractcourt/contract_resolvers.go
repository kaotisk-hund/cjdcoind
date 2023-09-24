package contractcourt

import (
	"encoding/binary"
	"io"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/channeldb"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

var (
	endian = binary.BigEndian
)

const (
	// sweepConfTarget is the default number of blocks that we'll use as a
	// confirmation target when sweeping.
	sweepConfTarget = 6
)

// ContractResolver is an interface which packages a state machine which is
// able to carry out the necessary steps required to fully resolve a Bitcoin
// contract on-chain. Resolvers are fully encodable to ensure callers are able
// to persist them properly. A resolver may produce another resolver in the
// case that claiming an HTLC is a multi-stage process. In this case, we may
// partially resolve the contract, then persist, and set up for an additional
// resolution.
type ContractResolver interface {
	// ResolverKey returns an identifier which should be globally unique
	// for this particular resolver within the chain the original contract
	// resides within.
	ResolverKey() []byte

	// Resolve instructs the contract resolver to resolve the output
	// on-chain. Once the output has been *fully* resolved, the function
	// should return immediately with a nil ContractResolver value for the
	// first return value.  In the case that the contract requires further
	// resolution, then another resolve is returned.
	//
	// NOTE: This function MUST be run as a goroutine.
	Resolve() (ContractResolver, er.R)

	// IsResolved returns true if the stored state in the resolve is fully
	// resolved. In this case the target output can be forgotten.
	IsResolved() bool

	// Encode writes an encoded version of the ContractResolver into the
	// passed Writer.
	Encode(w io.Writer) er.R

	// Stop signals the resolver to cancel any current resolution
	// processes, and suspend.
	Stop()
}

// htlcContractResolver is the required interface for htlc resolvers.
type htlcContractResolver interface {
	ContractResolver

	// HtlcPoint returns the htlc's outpoint on the commitment tx.
	HtlcPoint() wire.OutPoint

	// Supplement adds additional information to the resolver that is
	// required before Resolve() is called.
	Supplement(htlc channeldb.HTLC)
}

// reportingContractResolver is a ContractResolver that also exposes a report on
// the resolution state of the contract.
type reportingContractResolver interface {
	ContractResolver

	report() *ContractReport
}

// ResolverConfig contains the externally supplied configuration items that are
// required by a ContractResolver implementation.
type ResolverConfig struct {
	// ChannelArbitratorConfig contains all the interfaces and closures
	// required for the resolver to interact with outside sub-systems.
	ChannelArbitratorConfig

	// Checkpoint allows a resolver to check point its state. This function
	// should write the state of the resolver to persistent storage, and
	// return a non-nil error upon success. It takes a resolver report,
	// which contains information about the outcome and should be written
	// to disk if non-nil.
	Checkpoint func(ContractResolver, ...*channeldb.ResolverReport) er.R
}

// contractResolverKit is meant to be used as a mix-in struct to be embedded within a
// given ContractResolver implementation. It contains all the common items that
// a resolver requires to carry out its duties.
type contractResolverKit struct {
	ResolverConfig

	quit chan struct{}
}

// newContractResolverKit instantiates the mix-in struct.
func newContractResolverKit(cfg ResolverConfig) *contractResolverKit {
	return &contractResolverKit{
		ResolverConfig: cfg,
		quit:           make(chan struct{}),
	}
}

var (
	// errResolverShuttingDown is returned when the resolver stops
	// progressing because it received the quit signal.
	errResolverShuttingDown = er.GenericErrorType.CodeWithDetail("errResolverShuttingDown",
		"resolver shutting down")
)
