package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/simiotics/simplex/state"
)

func main() {
	var stateDir string

	simplexCommand := &cobra.Command{
		Use:              "simplex",
		Short:            "Single-node data processing with docker",
		Long:             "simplex lets you define data processing flows and then execute them using docker. It runs on a single machine.",
		TraverseChildren: true,
	}

	var stateCommand = &cobra.Command{
		Use:   "state",
		Short: "Interact with simplex state",
		Long:  "This command provides access to the simplex state database",
	}

	var initCommand = &cobra.Command{
		Use:   "init",
		Short: "Initializes a simplex state directory",
		Run: func(cmd *cobra.Command, args []string) {
			state.Init(stateDir)
		},
	}

	stateCommand.AddCommand(initCommand)

	simplexCommand.AddCommand(stateCommand)
	simplexCommand.Flags().StringVarP(&stateDir, "state-dir", "d", "~/.simplex", "Path to simplex state directory")

	err := simplexCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
