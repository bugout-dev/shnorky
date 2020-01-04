package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	simplexCmd := &cobra.Command{
		Use:   "simplex",
		Short: "Single-node data processing with docker",
		Long:  "simplex lets you define data processing flows and then execute them using docker. It runs on a single machine.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello")
		},
	}

	err := simplexCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
