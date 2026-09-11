package main

import (
	"fmt"
	"taskmanager/tasks"
)

func main() {
	mgr := tasks.NewManager()

	names := []string{"Fetch data", "Process file", "Send email", "Generate report", "Cleanup temp"}
	for i, name := range names {
		t, err := tasks.NewTask(i+1, name)
		if err != nil {
			fmt.Println("Error creating task:", err)
			continue
		}
		mgr.Add(t)
	}

	completed := mgr.RunAll(3) // 3 concurrent workers

	fmt.Println("\n--- Task Results ---")
	for _, t := range completed {
		fmt.Println(t)
	}

	fmt.Println("\n--- Summary ---")
	for status, count := range mgr.Summary() {
		fmt.Printf("%s: %d\n", status, count)
	}
}
