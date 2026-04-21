package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings"
	"github.com/ThreeDotsLabs/cli/trainings/git"
)

var (
	version = "dev"
	commit  = "n/a"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%v", v)
			}
			formatUnexpectedError(err)
			var st stackTracer
			if !errors.As(err, &st) {
				fmt.Println(color.HiBlackString("\nPanic stack:"))
				fmt.Println(color.HiBlackString(string(debug.Stack())))
			}
			os.Exit(1)
		}
	}()

	if version == "" || version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			version = strings.TrimPrefix(bi.Main.Version, "v")
		}
	}

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
			separator := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
			fmt.Println(separator)
			fmt.Println(color.RedString("ERROR: ") + userFacingErr.Msg)
			fmt.Println(separator)
			fmt.Println(color.GreenString("\nHow to solve: \n") + userFacingErr.SolutionHint)
			os.Exit(1)
			return
		}

		if err != nil {
			formatUnexpectedError(err)
		}

		os.Exit(1)
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			EnvVars: []string{"VERBOSE"},
		},
		&cli.BoolFlag{
			Name:   "force-update-prompt",
			Usage:  "force the update prompt to appear (for testing)",
			Hidden: true,
		},
	},
	Before: func(c *cli.Context) error {
		if verbose := c.Bool("verbose"); verbose {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.SetFormatter(&logrus.TextFormatter{})
		} else {
			logrus.SetLevel(logrus.WarnLevel)
		}

		commandName := ""
		for _, arg := range os.Args[1:] {
			if !strings.HasPrefix(arg, "-") {
				commandName = arg
				break
			}
		}

		internal.CheckForUpdate(version, commandName, c.Bool("force-update-prompt"))

		return nil
	},
	Commands: []*cli.Command{
		{
			Name:    "training",
			Aliases: []string{"tr"},
			Usage:   fmt.Sprintf("commands for %s commands", internal.WebsiteAddress),
			Before: func(c *cli.Context) error {
				sub := c.Args().First()
				if sub == "init" || sub == "configure" {
					return nil
				}
				return newHandlers(c).CheckServerConnection(c.Context, "", "", false)
			},
			Subcommands: []*cli.Command{
				{
					Name:      "configure",
					Usage:     "connect your environment with https://academy.threedots.tech/ account",
					ArgsUsage: fmt.Sprintf("<%s>", tokenDocs),
					Flags:     configureFlags,
					Before: func(c *cli.Context) error {
						return newHandlers(c).CheckServerConnection(
							c.Context,
							c.String("server"),
							c.String("region"),
							c.Bool("insecure"),
						)
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
					Flags: append(configureFlags,
						&cli.StringFlag{
							Name:  "token",
							Usage: tokenDocs,
						},
						&cli.BoolFlag{
							Name:  "no-git",
							Usage: "disable git integration",
						},
						&cli.BoolFlag{
							Name:   "force-git",
							Usage:  "enable git integration even in non-interactive mode",
							Hidden: true,
						},
					),
					Usage: "initialise training files in your current directory",
					Before: func(c *cli.Context) error {
						return newHandlers(c).CheckServerConnection(
							c.Context,
							c.String("server"),
							c.String("region"),
							c.Bool("insecure"),
						)
					},
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

						return handlers.Init(c.Context, trainingID, dir, c.Bool("no-git"), c.Bool("force-git"))
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
						&cli.IntFlag{
							Name:    "mcp-port",
							Usage:   "port for MCP server on 127.0.0.1 (0 to disable)",
							Value:   39131,
							EnvVars: []string{"TDL_MCP_PORT"},
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
					Name:  "rollback",
					Usage: "rollback to one of your past solutions for the current exercise",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Rollback(c.Context)
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
					Name:      "skip",
					Usage:     "Skip the current exercise and the rest of the module (only selected modules)",
					ArgsUsage: "",
					Action: func(c *cli.Context) error {
						return newHandlers(c).Skip(c.Context)
					},
				},
				{
					Name:  "settings",
					Usage: "View and change training settings",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "auto-commit", Usage: "auto-commit on exercise pass (on/off)"},
						&cli.StringFlag{Name: "auto-sync", Usage: "auto-sync with example solution (on/off)"},
						&cli.StringFlag{Name: "sync-mode", Usage: "sync mode: compare, merge, or override"},
						&cli.StringFlag{Name: "mcp", Usage: "MCP server for AI coding tools (on/off)"},
					},
					Action: func(c *cli.Context) error {
						var opts trainings.SettingsOptions
						if c.IsSet("auto-commit") {
							v := parseBoolFlag(c.String("auto-commit"))
							opts.AutoCommit = &v
						}
						if c.IsSet("auto-sync") {
							v := parseBoolFlag(c.String("auto-sync"))
							opts.AutoSync = &v
						}
						if c.IsSet("sync-mode") {
							v := c.String("sync-mode")
							opts.SyncMode = &v
						}
						if c.IsSet("mcp") {
							v := parseBoolFlag(c.String("mcp"))
							opts.MCP = &v
						}
						return newHandlers(c).Settings(opts)
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
		{
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "Update tdl to the latest version",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "yes",
					Aliases: []string{"y"},
					Usage:   "skip confirmation prompt",
				},
				&cli.StringFlag{
					Name:  "version",
					Usage: "update to a specific version or branch (e.g., v1.2.3, master)",
				},
			},
			Action: func(c *cli.Context) error {
				return internal.RunUpdate(c.Context, version, internal.UpdateOptions{
					SkipConfirm:   c.Bool("yes"),
					TargetVersion: c.String("version"),
				})
			},
		},
	},
}

func newHandlers(c *cli.Context) *trainings.Handlers {
	cmd := c.Command.HelpName
	if cmd == "" {
		cmd = c.Command.FullName()
	}

	mcpPort := c.Int("mcp-port")

	return trainings.NewHandlers(trainings.CliMetadata{
		Version:           version,
		Commit:            commit,
		Architecture:      runtime.GOARCH,
		OS:                runtime.GOOS,
		OSVersion:         osVersion(),
		GoVersion:         runtime.Version(),
		GitVersion:        gitVersionString(),
		ExecutedCommand:   cmd,
		Interactive:       internal.IsStdinTerminal(),
		ForceUpdatePrompt: c.Bool("force-update-prompt"),
	}, mcpPort)
}

func osVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseBoolFlag(s string) bool {
	switch strings.ToLower(s) {
	case "on", "true", "1", "yes":
		return true
	default:
		return false
	}
}

func gitVersionString() string {
	v, err := git.CheckVersion()
	if err != nil {
		return ""
	}
	return v.String()
}

type missingArgumentError struct {
	msg string
}

func (m missingArgumentError) Error() string {
	return m.msg
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func formatUnexpectedError(err error) {
	separator := color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))

	fmt.Println(separator)
	fmt.Println(color.RedString("ERROR: ") + err.Error())
	fmt.Println(separator)

	var st stackTracer
	if errors.As(err, &st) {
		fmt.Println(color.HiBlackString("\nStack trace:"))
		for _, frame := range st.StackTrace() {
			// %+v produces "funcName\n\tfile:line" — flatten to single line
			raw := fmt.Sprintf("%+v", frame)
			var parts []string
			for _, l := range strings.Split(raw, "\n") {
				if trimmed := strings.TrimSpace(l); trimmed != "" {
					parts = append(parts, trimmed)
				}
			}
			fmt.Println(color.HiBlackString("  " + strings.Join(parts, " ")))
		}
	}
}
