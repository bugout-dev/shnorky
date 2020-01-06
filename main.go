package main

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/spf13/cobra"

	"github.com/simiotics/simplex/state"
)

func main() {
	defaultStateDir := ".simplex"
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("Error looking up current user: %s", err.Error())
		os.Exit(1)
	}
	if currentUser.HomeDir != "" {
		defaultStateDir = path.Join(currentUser.HomeDir, defaultStateDir)
	}

	var stateDir string

	// simplex root command
	simplexCommand := &cobra.Command{
		Use:              "simplex",
		Short:            "Single-node data processing flows using docker",
		Long:             "simplex lets you define data processing flows and then execute them using docker. It runs on a single machine.",
		TraverseChildren: true,
	}

	// simplex completion
	completionCommand := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completions for the simplex command (for supported shells)",
	}

	bashCompletionCommand := &cobra.Command{
		Use:   "bash",
		Short: "bash completion for simplex",
		Long: `bash completion for simplex

If you are using bash and want command completion for the simplex CLI, run (ommiting the $):
	$ . <(simplex completion bash)
`,
		Run: func(cmd *cobra.Command, args []string) {
			simplexCommand.GenBashCompletion(os.Stdout)
		},
	}

	completionCommand.AddCommand(bashCompletionCommand)

	// simplex state
	stateCommand := &cobra.Command{
		Use:   "state",
		Short: "Interact with simplex state",
		Long:  "This command provides access to the simplex state database",
	}

	initCommand := &cobra.Command{
		Use:   "init",
		Short: "Initializes a simplex state directory",
		Run: func(cmd *cobra.Command, args []string) {
			state.Init(stateDir)
		},
	}

	stateCommand.AddCommand(initCommand)

	simplexCommand.AddCommand(completionCommand, stateCommand)
	simplexCommand.Flags().StringVar(&stateDir, "statedir", defaultStateDir, "Path to simplex state directory")

	err = simplexCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
