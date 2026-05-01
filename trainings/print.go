package trainings

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) printCurrentExercise(moduleName string, exerciseName string) {
	name := fmt.Sprintf("%v/%v", moduleName, exerciseName)
	fmt.Printf("\n%s\n", color.HiBlackString(strings.Repeat("─", internal.TerminalWidth())))
	fmt.Println(color.New(color.Bold, color.FgCyan).Sprint(name))
}

func (h *Handlers) printNotInATrainingDirectory() {
	fmt.Println("You are not in a training directory. If you already started the training, please go to the exercise directory.")
	fmt.Printf("Please run %s if you didn't start training yet.\n", internal.SprintCommand(internal.BinaryName()+" training init"))
}

func printFinished() {
	fmt.Println("Congratulations, you finished the training " + color.YellowString("🏆"))
}

func printCohortBatchDone(date *time.Time) {
	fmt.Println()
	fmt.Println("Good job, you're done with the current batch of modules! " + color.YellowString("✅"))
	fmt.Println()
	fmt.Println("Get some rest and come back later to continue the training.")

	if date != nil {
		fmt.Println("The next batch will be available on " + color.YellowString(date.Format("Monday Jan 2 2006")) + ".")
	}
}

func printPaymentRequired() {
	fmt.Println(color.GreenString("You finished the free part of the training. To continue, please go back to our website."))
}

func printTextOnlyExerciseInfo(trainingName, exerciseID string) {
	fmt.Println(
		color.GreenString("This lesson is text-only.\nYou can read it in your browser:"),
		internal.ExerciseURL(trainingName, exerciseID)+"\n",
	)
}

// textWidth returns the approximate display width of s in a terminal.
// Most characters are 1 column wide, but emoji like 💡 occupy 2 columns.
// We add +1 for characters in Unicode's Supplementary Multilingual Plane (≥ U+1F000)
// where most emoji live, to avoid a full East Asian Width table dependency.
func textWidth(s string) int {
	n := utf8.RuneCountInString(s)
	for _, r := range s {
		if r >= 0x1F000 {
			n++
		}
	}
	return n
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// printColorBox prints lines inside a dim-bordered box, preserving ANSI colors in content.
func printColorBox(lines ...string) {
	maxWidth := 0
	for _, line := range lines {
		if w := textWidth(stripANSI(line)); w > maxWidth {
			maxWidth = w
		}
	}

	border := strings.Repeat("─", maxWidth+3)
	dim := color.HiBlackString

	fmt.Println(dim("  ┌" + border + "┐"))
	for _, line := range lines {
		pad := strings.Repeat(" ", maxWidth-textWidth(stripANSI(line)))
		fmt.Printf("  %s  %s%s %s\n", dim("│"), line, pad, dim("│"))
	}
	fmt.Println(dim("  └" + border + "┘"))
}

// printDimBox prints lines inside a dim bordered box, indented by 2 spaces.
func printDimBox(lines ...string) {
	maxWidth := 0
	for _, line := range lines {
		if w := textWidth(line); w > maxWidth {
			maxWidth = w
		}
	}

	border := strings.Repeat("─", maxWidth+3)
	dim := color.HiBlackString

	fmt.Println(dim("  ┌" + border + "┐"))
	for _, line := range lines {
		pad := strings.Repeat(" ", maxWidth-textWidth(line))
		fmt.Println(dim("  │  " + line + pad + " │"))
	}
	fmt.Println(dim("  └" + border + "┘"))
}

func PrintScenarios(scenarios []*genproto.ScenarioResult) {
	fmt.Println()
	fmt.Println("--------")
	fmt.Println()

	for _, s := range scenarios {
		parts := strings.Split(s.Name, " / ")
		var name string
		if len(parts) > 1 {
			name = color.New(color.Bold).Sprint(strings.Join(parts[0:len(parts)-1], " / ")) + " / " + parts[len(parts)-1]
		} else {
			name = color.New(color.Bold).Sprint(parts[0])
		}

		if s.Failed {
			fmt.Println(color.RedString("✗") + " " + name)
		} else {
			fmt.Println(color.GreenString("✓") + " " + name)
		}

		if len(s.Logs) > 0 {
			lines := strings.Split(strings.TrimSpace(s.Logs), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					fmt.Println(line)
				}
			}
			fmt.Println()
		}
	}

	fmt.Println()
}
