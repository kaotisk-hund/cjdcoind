package main

import (
	"context"
	"fmt"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnrpc/lnclipb"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnrpc/verrpc"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinconfig/version"
	"github.com/urfave/cli"
)

var versionCommand = cli.Command{
	Name:  "version",
	Usage: "Display lncli and lnd version info.",
	Description: `
	Returns version information about both lncli and lnd. If lncli is unable
	to connect to lnd, the command fails but still prints the lncli version.
	`,
	Action: actionDecorator(v),
}

func v(ctx *cli.Context) er.R {
	conn := getClientConn(ctx, false)
	defer conn.Close()

	versions := &lnclipb.VersionResponse{
		Lncli: &verrpc.Version{
			Version:       version.Version(),
			AppMajor:      uint32(version.AppMajorVersion()),
			AppMinor:      uint32(version.AppMinorVersion()),
			AppPatch:      uint32(version.AppPatchVersion()),
			AppPreRelease: fmt.Sprintf("%v", version.IsPrerelease()),
		},
	}

	client := verrpc.NewVersionerClient(conn)

	ctxb := context.Background()
	lndVersion, err := client.GetVersion(ctxb, &verrpc.VersionRequest{})
	if err != nil {
		printRespJSON(versions)
		return er.Errorf("unable fetch version from lnd: %v", err)
	}
	versions.Lnd = lndVersion

	printRespJSON(versions)

	return nil
}
