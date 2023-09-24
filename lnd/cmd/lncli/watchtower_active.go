// +build watchtowerrpc

package main

import (
	"context"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnrpc/watchtowerrpc"
	"github.com/urfave/cli"
)

func watchtowerCommands() []cli.Command {
	return []cli.Command{
		{
			Name:     "tower",
			Usage:    "Interact with the watchtower.",
			Category: "Watchtower",
			Subcommands: []cli.Command{
				towerInfoCommand,
			},
		},
	}
}

func getWatchtowerClient(ctx *cli.Context) (watchtowerrpc.WatchtowerClient, func()) {
	conn := getClientConn(ctx, false)
	cleanup := func() {
		conn.Close()
	}
	return watchtowerrpc.NewWatchtowerClient(conn), cleanup
}

var towerInfoCommand = cli.Command{
	Name:   "info",
	Usage:  "Returns basic information related to the active watchtower.",
	Action: actionDecorator(towerInfo),
}

func towerInfo(ctx *cli.Context) er.R {
	if ctx.NArg() != 0 || ctx.NumFlags() > 0 {
		return er.E(cli.ShowCommandHelp(ctx, "info"))
	}

	client, cleanup := getWatchtowerClient(ctx)
	defer cleanup()

	req := &watchtowerrpc.GetInfoRequest{}
	resp, err := client.GetInfo(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)

	return nil
}
