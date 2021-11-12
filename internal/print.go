package internal

import "github.com/fatih/color"

func SprintCommand(cmd string) string {
	return color.CyanString(cmd)
}
