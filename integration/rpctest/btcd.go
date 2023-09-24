// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpctest

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

var (
	// compileMtx guards access to the executable path so that the project is
	// only compiled once.
	compileMtx sync.Mutex

	// executablePath is the path to the compiled executable. This is the empty
	// string until cjdcoind is compiled. This should not be accessed directly;
	// instead use the function cjdcoindExecutablePath().
	executablePath string
)

// cjdcoindExecutablePath returns a path to the cjdcoind executable to be used by
// rpctests. To ensure the code tests against the most up-to-date version of
// cjdcoind, this method compiles cjdcoind the first time it is called. After that, the
// generated binary is used for subsequent test harnesses. The executable file
// is not cleaned up, but since it lives at a static path in a temp directory,
// it is not a big deal.
func cjdcoindExecutablePath() (string, er.R) {
	compileMtx.Lock()
	defer compileMtx.Unlock()

	// If cjdcoind has already been compiled, just use that.
	if len(executablePath) != 0 {
		return executablePath, nil
	}

	testDir, err := baseDir()
	if err != nil {
		return "", err
	}

	// Build cjdcoind and output an executable in a static temp path.
	outputPath := filepath.Join(testDir, "cjdcoind")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}
	cmd := exec.Command(
		"go", "build", "-o", outputPath, "github.com/kaotisk-hund/cjdcoind",
	)
	err = er.E(cmd.Run())
	if err != nil {
		return "", er.Errorf("Failed to build cjdcoind: %v", err)
	}

	// Save executable path so future calls do not recompile.
	executablePath = outputPath
	return executablePath, nil
}
