package feature

import (
	"fmt"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
)

// ErrUnknownRequired signals that a feature vector requires certain features
// that our node is unaware of or does not implement.
var ErrUnknownRequired = er.GenericErrorType.CodeWithDetail("ErrUnknownRequired",
	"feature vector contains unknown required features")

// ValidateRequired returns an error if the feature vector contains a non-zero
// number of unknown, required feature bits.
func ValidateRequired(fv *lnwire.FeatureVector) er.R {
	unknown := fv.UnknownRequiredFeatures()
	if len(unknown) > 0 {
		return ErrUnknownRequired.New(fmt.Sprintf("%v", unknown), nil)
	}
	return nil
}
