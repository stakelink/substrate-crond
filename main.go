package main

import (
	"os"
	"log"
	"github.com/stakelink/substrate-crond/cron"

	"github.com/urfave/cli/v2"
)

func Run(ctx *cli.Context) error {
	c, err := cron.New(ctx.String("rpc-url"))
	if err != nil {
		return err
	}
	
	err = c.LoadCrontab(ctx.String("crontab-file"))
	
	if err != nil {
		return err
	}

  return c.Run()
}

func main() {
  app := &cli.App{
    Name: "substrate-crond",
    Usage: "Daemon to execute scheduled jobs based on Substrate activity",
    Action: Run,
    Flags: []cli.Flag {
    	&cli.StringFlag{
        	Name: "rpc-url",
        	Aliases: []string{"r"},
        	Value: "wss://rpc.polkadot.io",
        	Usage: "",
    	},
    	&cli.StringFlag{
        	Name: "crontab-file",
        	Aliases: []string{"c"},
        	Value: "/etc/substrate-crond/crontab",
        	Usage: "",
    	},
    	&cli.BoolFlag{
        	Name: "daemon",
        	Aliases: []string{"d"},
        	Value: false,
        	Usage: "",
    	},
    },
  }

  err := app.Run(os.Args)
  if err != nil {
    log.Fatal(err)
  }
}
