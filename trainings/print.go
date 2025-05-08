package trainings

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/ThreeDotsLabs/cli/trainings/genproto"
)

func (h *Handlers) printCurrentExercise(moduleName string, exerciseName string) {
	name := fmt.Sprintf("%v/%v", moduleName, exerciseName)
	fmt.Printf("\n%s\n", color.New(color.Bold, color.FgCyan).Sprint(name))
}

func (h *Handlers) printNotInATrainingDirectory() {
	fmt.Println("You are not in a training directory. If you already started the training, please go to the exercise directory.")
	fmt.Printf("Please run %s if you didn't start training yet.\n", internal.SprintCommand("tdl training init"))
}

func printFinished() {
	fmt.Println("Congratulations, you finished the training " + color.YellowString("🏆"))
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
