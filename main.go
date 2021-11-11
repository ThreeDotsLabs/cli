package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ThreeDotsLabs/cli/tdl/trainings"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "tdl",
		Usage:     "https://threedots.tech/ CLI.",
		Compiled:  time.Now(),
		Copyright: "(c) Three Dots Labs",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				EnvVars: []string{"DEBUG"},
			},
		},
		Before: func(c *cli.Context) error {
			if debug := c.Bool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.SetFormatter(&logrus.TextFormatter{})
			} else {
				logrus.SetOutput(io.Discard)
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "training",
				Aliases: []string{"tr"},
				Usage:   "commands for https://learn.threedots.tech/ commands",
				Subcommands: []*cli.Command{
					{
						Name:      "configure",
						Usage:     "connect your environment with platform account",
						ArgsUsage: "[token from https://learn.threedots.tech/]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:   "server",
								Usage:  "custom server",
								Hidden: true,
							},
							&cli.BoolFlag{
								Name:  "override",
								Usage: "if config already exists, it will be overridden",
							},
						},
						Action: func(c *cli.Context) error {
							token := c.Args().First()

							if token == "" {
								fmt.Print("Missing token argument! Usage:\n\n")
								cli.ShowSubcommandHelpAndExit(c, 1)
							}

							return trainings.ConfigureGlobally(token, c.String("server"), c.Bool("override"))
						},
					},
					{
						Name:  "list",
						Usage: "list trainings",
						Action: func(c *cli.Context) error {
							return trainings.List()
						},
					},
					{
						Name:    "run",
						Aliases: []string{"r"},
						Usage:   "run exercise",
						Action: func(c *cli.Context) error {
							return trainings.Run()
						},
					},
					{
						Name:      "init",
						ArgsUsage: "[trainingID]",
						Usage:     "initialise training files in your current directory",
						Action: func(c *cli.Context) error {
							trainingID := c.Args().First()

							if trainingID == "" {
								fmt.Print("Missing trainingID argument! Usage:\n\n")
								cli.ShowSubcommandHelpAndExit(c, 1)
							}

							return trainings.Init(trainingID)
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
