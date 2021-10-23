package main

import (
	"errors"
	"io"
	"log"
	"os"
	"time"

	"github.com/ThreeDotsLabs/cli/tdl/course"
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
								return errors.New("missing token argument")
							}

							return course.ConfigureGlobally(token, c.String("server"), c.Bool("override"))
						},
					},
					{
						Name:  "list",
						Usage: "list trainings",
						Action: func(c *cli.Context) error {
							return course.List()
						},
					},
					{
						Name:    "run",
						Aliases: []string{"r"},
						Usage:   "run exercise",
						Action: func(c *cli.Context) error {
							return course.Run()
						},
					},
					{
						Name:      "init",
						ArgsUsage: "[training id]",
						Usage:     "initialise training files in your current directory",
						Action: func(c *cli.Context) error {
							trainingID := c.Args().First()
							if trainingID == "" {
								return errors.New("missing trainingID argument")
							}

							return course.Init(trainingID)
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
