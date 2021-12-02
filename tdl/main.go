package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/ThreeDotsLabs/cli/trainings"
	"github.com/ThreeDotsLabs/cli/trainings/web"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

var app = &cli.App{
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
			logrus.SetLevel(logrus.WarnLevel)
		}
		return nil
	},
	Commands: []*cli.Command{
		{
			Name:    "training",
			Aliases: []string{"tr"},
			Usage:   fmt.Sprintf("commands for %s commands", web.Website),
			Subcommands: []*cli.Command{
				{
					Name:      "configure",
					Usage:     "connect your environment with platform account",
					ArgsUsage: fmt.Sprintf("[token from %s]", web.Website),
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:   "server",
							Usage:  "custom server",
							Hidden: true,
						},
						&cli.BoolFlag{
							Name:   "insecure",
							Usage:  "do not verify certificate",
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

						return trainings.NewHandlers().ConfigureGlobally(
							c.Context,
							token,
							c.String("server"),
							c.Bool("override"),
							c.Bool("insecure"),
						)
					},
				},
				{
					Name:  "list",
					Usage: "list training",
					Action: func(c *cli.Context) error {
						return trainings.NewHandlers().List(c.Context)
					},
				},
				{
					Name:    "run",
					Aliases: []string{"r"},
					Usage:   "run exercise",
					Action: func(c *cli.Context) error {
						success, err := trainings.NewHandlers().Run(c.Context)
						if err != nil {
							return err
						}

						if !success {
							os.Exit(1)
						}
						return nil
					},
				},
				{
					Name:      "init",
					ArgsUsage: fmt.Sprintf("[trainingID from %s]", web.Website),
					Usage:     "initialise training files in your current directory",
					Action: func(c *cli.Context) error {
						trainingID := c.Args().First()

						if trainingID == "" {
							return missingArgumentError{"Missing trainingID argument"}
						}

						return trainings.NewHandlers().Init(c.Context, trainingID)
					},
				},
			},
		},
	},
}

type missingArgumentError struct {
	msg string
}

func (m missingArgumentError) Error() string {
	return m.msg
}
