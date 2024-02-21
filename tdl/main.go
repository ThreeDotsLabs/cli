package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings"
)

var (
	version = "dev"
	commit  = "n/a"
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

		userFacingErr := trainings.UserFacingError{}
		if errors.As(err, &userFacingErr) {
			fmt.Printf(color.RedString("ERROR: ") + userFacingErr.Msg + "\n")
			fmt.Printf(color.GreenString("\nHow to solve: \n") + userFacingErr.SolutionHint + "\n")
			os.Exit(1)
			return
		}

		if err != nil {
			fmt.Printf("%+v\n", err)
		}

		os.Exit(1)
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			EnvVars: []string{"VERBOSE"},
		},
	},
	Before: func(c *cli.Context) error {
		if verbose := c.Bool("verbose"); verbose {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.SetFormatter(&logrus.TextFormatter{})
		} else {
			logrus.SetLevel(logrus.WarnLevel)
		}

		internal.CheckForUpdate(version)

		return nil
	},
	Commands: []*cli.Command{
		{
			Name:    "training",
			Aliases: []string{"tr"},
			Usage:   fmt.Sprintf("commands for %s commands", internal.WebsiteAddress),
			Subcommands: []*cli.Command{
				{
					Name:      "configure",
					Usage:     "connect your environment with platform account",
					ArgsUsage: fmt.Sprintf("[token from %s]", internal.WebsiteAddress),
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:   "server",
							Usage:  "custom server",
							Hidden: true,
						},
						&cli.StringFlag{
							Name:   "region",
							Usage:  "the region to use (eu or us)",
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

						return newHandlers(c).ConfigureGlobally(
							c.Context,
							token,
							c.String("server"),
							c.String("region"),
							c.Bool("override"),
							c.Bool("insecure"),
						)
					},
				},
				{
					Name:      "init",
					ArgsUsage: fmt.Sprintf("[trainingID from %s]", internal.WebsiteAddress),
					Usage:     "initialise training files in your current directory",
					Action: func(c *cli.Context) error {
						trainingID := c.Args().First()

						if trainingID == "" {
							return missingArgumentError{"Missing trainingID argument"}
						}

						return newHandlers(c).Init(c.Context, trainingID)
					},
				},
				{
					Name:  "reset-exercise",
					Usage: "Reset exercise downloads files for the current exercise again",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Reset(c.Context)
					},
				},
				{
					Name:    "run",
					Aliases: []string{"r"},
					Usage:   "run exercise",
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name:    "detached",
							Aliases: []string{"d"},
							Usage:   "running in non-interactive mode",
						},
					},
					Action: func(c *cli.Context) error {
						err := newHandlers(c).Run(c.Context, c.Bool("detached"))
						if err != nil {
							return err
						}

						return nil
					},
				},
				{
					Name:    "info",
					Aliases: []string{"i"},
					Usage:   "print information about current training",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Info(c.Context)
					},
				},
				{
					Name:  "list",
					Usage: "list training",
					Action: func(c *cli.Context) error {
						return newHandlers(c).List(c.Context)
					},
				},
			},
		},
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Prints version of TDL CLI",
			Action: func(c *cli.Context) error {
				fmt.Println("Version:", version)
				fmt.Println("Commit:", commit)
				fmt.Println("Architecture:", runtime.GOARCH)
				fmt.Println("OS:", runtime.GOOS)
				return nil
			},
		},
	},
}

func newHandlers(c *cli.Context) *trainings.Handlers {
	return trainings.NewHandlers(trainings.CliMetadata{
		Version:         version,
		Commit:          commit,
		Architecture:    runtime.GOARCH,
		OS:              runtime.GOOS,
		ExecutedCommand: c.Command.HelpName,
	})
}

type missingArgumentError struct {
	msg string
}

func (m missingArgumentError) Error() string {
	return m.msg
}
