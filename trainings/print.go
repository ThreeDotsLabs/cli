package trainings

import (
	"fmt"

	"github.com/fatih/color"

	"github.com/ThreeDotsLabs/cli/internal"
)

func (h *Handlers) printCurrentExercise(moduleName string, exerciseName string) {
	name := fmt.Sprintf("%v/%v", moduleName, exerciseName)
	fmt.Printf("\n%s\n", color.New(color.Bold, color.FgCyan).Sprint(name))
}

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

func printPaymentRequired() {
	fmt.Println(color.GreenString("You finished the free part of the training. To continue, please go back to our website."))
}

func printTextOnlyExerciseInfo(trainingName, exerciseID string) {
	fmt.Println(
		color.GreenString("This lesson is text-only.\nYou can read it in your browser:"),
		internal.ExerciseURL(trainingName, exerciseID)+"\n",
	)
}
