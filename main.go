package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"sync"

	docker "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/simiotics/simplex/components"
	"github.com/simiotics/simplex/state"
)

// logLevels - mapping between log level specification strings and logrus Level values
var logLevels = map[string]logrus.Level{
	"TRACE": logrus.TraceLevel,
	"DEBUG": logrus.DebugLevel,
	"INFO":  logrus.InfoLevel,
	"WARN":  logrus.WarnLevel,
	"ERROR": logrus.ErrorLevel,
	"FATAL": logrus.FatalLevel,
	"PANIC": logrus.PanicLevel,
}

// Accepts the following environment variables:
// + LOG_LEVEL (value should be one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC)
func generateLogger() *logrus.Logger {
	log := logrus.New()

	rawLevel := os.Getenv("LOG_LEVEL")
	if rawLevel == "" {
		rawLevel = "WARN"
	}
	level, ok := logLevels[rawLevel]
	if !ok {
		log.Fatalf("Invalid value for LOG_LEVEL environment variable: %s. Choose one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC", rawLevel)
	}
	log.SetLevel(level)

	return log
}

// Version denotes the current version of the simplex tool and library
var Version = "0.1.0-dev"

var log = generateLogger()

func openStateDB(stateDir string) *sql.DB {
	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		log.WithFields(logrus.Fields{"stateDBPath": stateDBPath, "error": err}).Fatal("Error opening state database")
	}
	return db
}

func generateDockerClient() *docker.Client {
	client, err := docker.NewEnvClient()
	if err != nil {
		log.WithField("error", err).Fatal("Error creating docker client")
	}
	return client
}

func main() {
	defaultStateDir := ".simplex"
	currentUser, err := user.Current()
	if err != nil {
		log.WithField("error", err).Fatal("Error looking up current user")
	}
	if currentUser.HomeDir != "" {
		defaultStateDir = path.Join(currentUser.HomeDir, defaultStateDir)
	}

	var id, componentType, componentPath, specificationPath, stateDir string

	simplexCommand := &cobra.Command{
		Use:              "simplex",
		Short:            "Single-node data processing flows using docker",
		Long:             "simplex lets you define data processing flows and then execute them using docker. It runs on a single machine.",
		TraverseChildren: true,
	}

	simplexCommand.PersistentFlags().StringVarP(&stateDir, "statedir", "S", defaultStateDir, "Path to simplex state directory")

	// simplex version
	versionCommand := &cobra.Command{
		Use:   "version",
		Short: "simplex version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
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
			logger := log.WithField("stateDir", stateDir)
			logger.Info("Initializing state directory")
			err := state.Init(stateDir)
			if err != nil {
				logger.WithField("error", err).Fatal("Initialization failed")
			}
			logger.Info("Done")
			fmt.Println(stateDir)
		},
	}

	stateCommand.AddCommand(initCommand)

	// simplex components
	componentsCommand := &cobra.Command{
		Use:   "components",
		Short: "Interact with simplex components",
		Long: `Interact with simplex components

simplex components represent individual steps in a data processing flow. This command allows you
to interact with your simplex components (add new components, inspect existing components, and
remove unwanted components from your simplex state).
`,
	}

	addComponentCommand := &cobra.Command{
		Use:   "create",
		Short: "Add a component to simplex",
		Long:  "Adds a new component to simplex and makes it available in the state database",
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.WithFields(
				logrus.Fields{
					"id":                id,
					"componentType":     componentType,
					"componentPath":     componentPath,
					"specificationPath": specificationPath,
					"stateDir":          stateDir,
				},
			)

			logger.Debug("Opening state directory")
			db := openStateDB(stateDir)
			defer db.Close()

			logger.Debug("Adding component to state database")
			component, err := components.AddComponent(db, id, componentType, componentPath, specificationPath)
			if err != nil {
				logger.WithField("error", err).Fatal("Failed to add component")
			}
			logger.Info("Component added successfully")

			marshalledComponent, err := json.Marshal(component)
			if err != nil {
				logger.Fatal("Failed to marshall added component")
			}
			fmt.Println(string(marshalledComponent))
		},
	}

	addComponentCommand.Flags().StringVarP(&id, "id", "i", "", "ID for the component being added")

	componentTypesHelp := fmt.Sprintf("Type of component being added (one of: %s)", strings.Join([]string{components.Service, components.Task}, ","))
	addComponentCommand.Flags().StringVarP(&componentType, "type", "t", "", componentTypesHelp)

	addComponentCommand.Flags().StringVarP(&componentPath, "component", "c", "", "Directory in which component is defined")

	addComponentCommand.Flags().StringVarP(&specificationPath, "spec", "s", "", "Path to component specification")

	listComponentsCommand := &cobra.Command{
		Use:   "list",
		Short: "List all components registered against the state database",
		Long:  "Lists all components that have previously been added to the state database",
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			componentsChan := make(chan components.ComponentMetadata)
			db := openStateDB(stateDir)
			defer db.Close()

			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					enc := json.NewEncoder(os.Stdout)
					component, ok := <-componentsChan
					if !ok {
						return
					}
					err := enc.Encode(component)
					if err != nil {
						log.WithField("component", component).WithField("error", err).Error("Error marshalling component")
					}
				}
			}()

			err := components.ListComponents(db, componentsChan)
			if err != nil {
				log.WithField("error", err).Fatal("Could not list components")
			}
			wg.Wait()

			log.Info("ListComponents done")
		},
	}

	removeComponentCommand := &cobra.Command{
		Use:   "remove",
		Short: "Remove a component from simplex",
		Long:  "Removes a component registered against simplex from the state database",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()
			err := components.RemoveComponent(db, id)
			if err != nil {
				log.WithField("error", err).Errorf("Error removing component: %s", err.Error())
			}
			fmt.Println(id)
			log.Info("RemoveComponent done")
		},
	}

	removeComponentCommand.Flags().StringVarP(&id, "id", "i", "", "ID for the component being removed")

	componentsCommand.AddCommand(addComponentCommand, listComponentsCommand, removeComponentCommand)

	// simplex builds
	buildsCommand := &cobra.Command{
		Use:   "builds",
		Short: "Interact with simplex builds",
		Long: `Interact with simplex builds

simplex builds are images that can be used to execute simplex components inside containers. A build
derives from a component.
`,
	}

	createBuildCommand := &cobra.Command{
		Use:   "create",
		Short: "Create a build for a specific component",
		Long:  "Creates an image for the specified component using its current state on the filesystem",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			_, err := components.CreateBuild(ctx, db, dockerClient, os.Stdout, id)
			if err != nil {
				log.WithField("error", err).Fatal("Could not create build")
			}
		},
	}

	createBuildCommand.Flags().StringVarP(&id, "component", "c", "", "ID of the component for which build is being created")

	listBuildsCommand := &cobra.Command{
		Use:   "list",
		Short: "List builds registered against the state database",
		Long:  "Lists builds that have previously been added to the state database (allows listing by component ID)",
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.WithField("component", id)

			var wg sync.WaitGroup
			buildsChan := make(chan components.BuildMetadata)
			db := openStateDB(stateDir)
			defer db.Close()

			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					enc := json.NewEncoder(os.Stdout)
					build, ok := <-buildsChan
					if !ok {
						return
					}
					err := enc.Encode(build)
					if err != nil {
						logger.WithField("build", build).WithField("error", err).Error("Error marshalling build")
					}
				}
			}()

			err := components.ListBuilds(db, buildsChan, id)
			if err != nil {
				logger.WithField("error", err).Fatal("Could not list builds")
			}
			wg.Wait()

			logger.Info("ListBuilds done")
		},
	}

	listBuildsCommand.Flags().StringVarP(&id, "component", "c", "", "ID of the component for which builds are being created (optional; if not set, lists all builds)")

	buildsCommand.AddCommand(createBuildCommand, listBuildsCommand)

	// simplex executions
	executionsCommand := &cobra.Command{
		Use:   "executions",
		Short: "Interact with simplex executions",
		Long: `Interact with simplex executions

simplex executions are tasks or services created run from simplex components. Each execution is
associated with a single build, and any configuration it requires is specified in the component
that the build represents.
`,
	}

	// TODO(nkashy1): Accept mounts from command line (as a JSON string?)
	createExecutionCommand := &cobra.Command{
		Use:   "create",
		Short: "Execute a build for a specific component",
		Long:  "Creates a container for the given build and registers the container in the state database",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			executionMetadata, err := components.Execute(ctx, db, dockerClient, id, "", nil)
			if err != nil {
				log.WithField("error", err).Fatal("Could not execute build")
			}

			fmt.Println(executionMetadata.ID)
		},
	}

	createExecutionCommand.Flags().StringVarP(&id, "build", "b", "", "ID of the build being executed")

	executionsCommand.AddCommand(createExecutionCommand)

	simplexCommand.AddCommand(versionCommand, completionCommand, stateCommand, componentsCommand, buildsCommand, executionsCommand)

	err = simplexCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
