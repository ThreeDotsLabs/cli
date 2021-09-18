package main

import (
	"io"

	"github.com/ThreeDotsLabs/cli/course"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use: "tdl",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.SetFormatter(&logrus.TextFormatter{})
			} else {
				logrus.SetOutput(io.Discard)
			}
		},
	}
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "")

	var cmdPrint = &cobra.Command{
		Use:   "course",
		Short: "Print anything to the screen", // todo
	}

	cmdPrint.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "todo", // todo
		Run: func(cmd *cobra.Command, args []string) {
			course.Run()
		},
	})

	cmdPrint.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "todo", // todo
		Run: func(cmd *cobra.Command, args []string) {
			course.Start()
		},
	})

	rootCmd.AddCommand(cmdPrint)

	if err := rootCmd.Execute(); err != nil {
		// todo - no panic?
		panic(err)
	}
}
