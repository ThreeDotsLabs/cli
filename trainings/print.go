package trainings

import (
	"fmt"

	"github.com/ThreeDotsLabs/cli/internal"
	"github.com/fatih/color"
)

func (h *Handlers) printNotInATrainingDirectory() {
	fmt.Println("You are not in a training directory. If you already started the training, please go to the exercise directory.")
	fmt.Printf("Please run %s if you didn't start training yet.\n", internal.SprintCommand("tdl training init"))
}

func (h *Handlers) printExerciseTips() {
	fmt.Printf("To run solution, please execute " + internal.SprintCommand("tdl training run"))
	fmt.Println()
}

func printFinished() {
	fmt.Println("Congratulations, you finished the training " + color.YellowString("🏆"))
}
