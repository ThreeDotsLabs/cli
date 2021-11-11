package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ThreeDotsLabs/cli/tdl/trainings"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "tdl",
		Usage:     "https://threedots.tech/ CLI.",
		Compiled:  time.Now(),
		Copyright: "(c) Three Dots Labs",
		ExitErrHandler: func(c *cli.Context, err error) {
			if errors.As(err, &missingArgumentError{}) {
				fmt.Printf("%s. Usage:\n\n", err.Error())
				cli.ShowSubcommandHelpAndExit(c, 1)
			}

			fmt.Printf("%+v\n", err)
			os.Exit(1)

		},
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
				logrus.SetLevel(logrus.ErrorLevel)
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
								return missingArgumentError{"Missing token argument"}
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
								return missingArgumentError{"Missing trainingID argument"}
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

type missingArgumentError struct {
	msg string
}

func (m missingArgumentError) Error() string {
	return m.msg
}
