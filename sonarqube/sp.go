package main

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

func showSpinnerWithSteps() {
	// Create a new spinner
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)

	// Set the spinner color
	s.Color("cyan")

	// Set the spinner prefix
	s.Prefix = "✅ Step"

	// Start the spinner
	s.Start()

	// Simulate different steps
	for i := 1; i <= 5; i++ {
		// Perform some work or task here for each step
		time.Sleep(1 * time.Second)

		// Update the spinner text
		s.Suffix = fmt.Sprintf(" %d of 5", i)
	}

	// Stop the spinner when done
	s.Stop()
}
func main() {
	fmt.Println("Running steps with spinner:")
	showSpinnerWithSteps()
	fmt.Println("Steps completed.")
}

