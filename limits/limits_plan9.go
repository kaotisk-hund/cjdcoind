// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package limits

import "github.com/kaotisk-hund/cjdcoind/btcutil/er"

// SetLimits is a no-op on Plan 9 due to the lack of process accounting.
func SetLimits() er.R {
	return nil
}
