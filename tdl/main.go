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

var configureFlags = []cli.Flag{
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
	&cli.StringFlag{
		Name:  "region",
		Usage: "the region to use (eu or us)",
	},
	&cli.BoolFlag{
		Name:  "override",
		Usage: "if config already exists, it will be overridden",
		// deprecated, backward compatibility
		Hidden: true,
	},
}

var tokenDocs = fmt.Sprintf("token from %s", internal.WebsiteAddress)

var app = &cli.App{
	Name:      "tdl",
	Usage:     "https://threedots.tech/ CLI.",
	Compiled:  time.Now(),
	Copyright: "(c) Three Dots Labs",
	ExitErrHandler: func(c *cli.Context, err error) {
		if errors.As(err, &missingArgumentError{}) {
			fmt.Printf("%s. Usage:\n\n", err.Error())
			cli.ShowSubcommandHelpAndExit(c, 1)
			return
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
					Usage:     "connect your environment with https://academy.threedots.tech/ account",
					ArgsUsage: fmt.Sprintf("<%s>", tokenDocs),
					Flags:     configureFlags,
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
							c.Bool("insecure"),
						)
					},
				},
				{
					Name: "init",
					ArgsUsage: fmt.Sprintf(
						"<trainingID from %s> [directory, if empty defaults to trainingID]",
						internal.WebsiteAddress,
					),
					Flags: append(configureFlags, &cli.StringFlag{
						Name:  "token",
						Usage: tokenDocs,
					}),
					Usage: "initialise training files in your current directory",
					Action: func(c *cli.Context) error {
						trainingID := c.Args().First()

						if trainingID == "" {
							return missingArgumentError{"Missing trainingID argument"}
						}

						dir := c.Args().Get(1)
						if dir == "" {
							dir = trainingID
						}

						handlers := newHandlers(c)

						if c.String("token") != "" {
							err := handlers.ConfigureGlobally(
								c.Context,
								c.String("token"),
								c.String("server"),
								c.String("region"),
								c.Bool("insecure"),
							)
							if err != nil {
								return fmt.Errorf("could not configure training: %w", err)
							}
						}

						return handlers.Init(c.Context, trainingID, dir)
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
				{
					Name:  "checkout",
					Usage: "checkout one of your past solutions for the current exercise",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Checkout(c.Context)
					},
				},
				{
					Name:  "clone",
					Usage: "clone solution files to current directory",
					ArgsUsage: fmt.Sprintf(
						"<executionID from 'Share your solution' at %s> [directory, if empty defaults to current directory]",
						internal.WebsiteAddress,
					),
					Action: func(c *cli.Context) error {
						executionID := c.Args().First()
						if executionID == "" {
							return missingArgumentError{"Missing executionID argument"}
						}

						directory := c.Args().Get(1)

						return newHandlers(c).Clone(c.Context, executionID, directory)
					},
				},
				{
					Name:  "jump",
					Usage: "Jump to the exercise to work on. Provide the ID or keep empty for interactive mode",
					UsageText: `Provide one of:
  - exercise ID (e.g., 48cfc4c8-ceab-4438-8082-9ec6e322df58)
  - exercise name with module (e.g., 04-module/05-exercise)
  - exercise name, assuming current module (e.g., 05-exercise)
  - just the numbers (e.g., 4/5, or 5)
  - latest - to go back to the last exercise

Leave empty to pick interactively.

Note: after completing this exercise, the next exercise will be the last one you didn't complete yet'.`,

					ArgsUsage: fmt.Sprintf(
						"[exerciseID or name]",
					),
					Action: func(c *cli.Context) error {
						handlers := newHandlers(c)
						var err error

						exerciseID := c.Args().First()
						if exerciseID == "" {
							exerciseID, err = handlers.SelectExercise(c.Context)
							if err != nil {
								return err
							}
						} else {
							exerciseID, err = handlers.FindExercise(c.Context, exerciseID)
							if err != nil {
								return err
							}
						}

						if exerciseID == "" {
							return nil
						}

						return handlers.Jump(c.Context, exerciseID)
					},
				},
				{
					Name:  "restore",
					Usage: "Restore your latest solution files. Useful when starting from scratch on another machine",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Restore(c.Context)
					},
				},
				{
					Name:      "skip",
					Usage:     "Skip the current exercise and the rest of the module (only selected modules)",
					ArgsUsage: "",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Skip(c.Context)
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
