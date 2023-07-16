package main

import (
	"log"
	"os"
	"ruyka/pkg/config"
	"ruyka/pkg/version"

	"github.com/urfave/cli"
)

func main() {
	ruyka := cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "development",
				EnvVar:   "RUYKA_DEVELOPMENT",
				Required: false,
			},
		},
		Action:   run,
		Version:  version.Version,
		Commands: []cli.Command{},
	}

	if err := ruyka.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(cxt *cli.Context) error {
	c := config.New()
	if cxt.Bool("development") {
		c.DevMode()
	}

	server, err := c.Build()
	if err != nil {
		return err
	}
	go server.Run()
	return server.Shutdown()
}
