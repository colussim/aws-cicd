package main

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

func showSpinnerWithSteps() {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("cyan")
	s.Prefix = "Step"
	s.Start()

	for i := 1; i <= 5; i++ {
		// Print the step information on a new line
		fmt.Printf("\r%s %s of 5\n", s.Prefix, "ns")
fmt.Printf("✅ Namespace created successfully\n")

		// Perform some work or task here for each step
		time.Sleep(1 * time.Second)

		// Update the spinner text
//		s.Suffix = fmt.Sprintf(" %d of 5", i)
	}

	s.Stop()
}

func main() {
	fmt.Println("Running steps with spinner:")
	showSpinnerWithSteps()
	fmt.Println("Steps completed.")
}

