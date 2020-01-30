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

	"github.com/simiotics/shnorky/components"
	"github.com/simiotics/shnorky/flows"
	"github.com/simiotics/shnorky/state"
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

// Version denotes the current version of the shnorky tool and library
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
	defaultStateDir := ".shn"
	currentUser, err := user.Current()
	if err != nil {
		log.WithField("error", err).Fatal("Error looking up current user")
	}
	if currentUser.HomeDir != "" {
		defaultStateDir = path.Join(currentUser.HomeDir, defaultStateDir)
	}

	var id, componentType, componentPath, specificationPath, stateDir, mountConfig string

	shnorkyCommand := &cobra.Command{
		Use:              "shn",
		Short:            "Shnorky: Single-machine data processing flows using docker",
		Long:             "shnorky lets you define data processing flows and then execute them using docker. It runs on a single machine.",
		TraverseChildren: true,
	}

	shnorkyCommand.PersistentFlags().StringVarP(&stateDir, "statedir", "S", defaultStateDir, "Path to shnorky state directory")

	// shnorky version
	versionCommand := &cobra.Command{
		Use:   "version",
		Short: "shnorky version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	// shnorky completion
	completionCommand := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completions for the shnorky command (for supported shells)",
	}

	bashCompletionCommand := &cobra.Command{
		Use:   "bash",
		Short: "bash completion for shnorky",
		Long: `bash completion for shnorky

If you are using bash and want command completion for the shnorky CLI, run (ommiting the $):
	$ . <(shnorky completion bash)
`,
		Run: func(cmd *cobra.Command, args []string) {
			shnorkyCommand.GenBashCompletion(os.Stdout)
		},
	}

	completionCommand.AddCommand(bashCompletionCommand)

	// shnorky state
	stateCommand := &cobra.Command{
		Use:   "state",
		Short: "Interact with shnorky state",
		Long:  "This command provides access to the shnorky state database",
	}

	initCommand := &cobra.Command{
		Use:   "init",
		Short: "Initializes a shnorky state directory",
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

	// shnorky components
	componentsCommand := &cobra.Command{
		Use:   "components",
		Short: "Interact with shnorky components",
		Long: `Interact with shnorky components

shnorky components represent individual steps in a data processing flow. This command allows you
to interact with your shnorky components (add new components, inspect existing components, remove
unwanted components from your shnorky state, and build and execute components).
`,
	}

	createComponentCommand := &cobra.Command{
		Use:   "create",
		Short: "Add a component to shnorky",
		Long:  "Adds a new component to shnorky and makes it available in the state database",
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

			logger.Debug("Opening state database")
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

	createComponentCommand.Flags().StringVarP(&id, "id", "i", "", "ID for the component being added")

	componentTypesHelp := fmt.Sprintf("Type of component being added (one of: %s)", strings.Join([]string{components.Service, components.Task}, ","))
	createComponentCommand.Flags().StringVarP(&componentType, "type", "t", "", componentTypesHelp)

	createComponentCommand.Flags().StringVarP(&componentPath, "component", "c", "", "Directory in which component is defined")

	createComponentCommand.Flags().StringVarP(&specificationPath, "spec", "s", "", "Path to component specification")

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
		Short: "Remove a component from shnorky",
		Long:  "Removes a component registered against shnorky from the state database",
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

	createBuildCommand := &cobra.Command{
		Use:   "build",
		Short: "Create a build for a specific component",
		Long:  "Creates an image for the specified component using its current state on the filesystem",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			buildMetadata, err := components.CreateBuild(ctx, db, dockerClient, os.Stdout, id)
			if err != nil {
				log.WithField("error", err).Fatal("Could not create build")
			}
			fmt.Println("Build succeeded:", buildMetadata.ID)
		},
	}

	createBuildCommand.Flags().StringVarP(&id, "id", "i", "", "ID of the component for which build is being created")

	listBuildsCommand := &cobra.Command{
		Use:   "list-builds",
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

	listBuildsCommand.Flags().StringVarP(&id, "id", "i", "", "ID of the component for which builds are being listed (optional; if not set, lists all builds)")

	createExecutionCommand := &cobra.Command{
		Use:   "execute",
		Short: "Execute a build for a specific component",
		Long:  "Creates a container for the given build and registers the container in the state database",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			mounts, err := components.ReadMountConfiguration(strings.NewReader(mountConfig))
			if err != nil {
				log.WithField("error", err).Fatal("Error reading mount configuration")
			}

			executionMetadata, err := components.Execute(ctx, db, dockerClient, id, "", mounts)
			if err != nil {
				log.WithField("error", err).Fatal("Could not execute build")
			}

			fmt.Println(executionMetadata.ID)
		},
	}

	createExecutionCommand.Flags().StringVarP(&id, "build", "b", "", "ID of the build being executed")
	createExecutionCommand.Flags().StringVarP(&mountConfig, "mounts", "m", "", "JSON string specifying mount configuration for execution")

	componentsCommand.AddCommand(
		createComponentCommand,
		listComponentsCommand,
		removeComponentCommand,
		createBuildCommand,
		listBuildsCommand,
		createExecutionCommand,
	)

	// shnorky flows
	flowsCommand := &cobra.Command{
		Use:   "flows",
		Short: "Interact with shnorky flows",
		Long: `Interact with shnorky flows

shnorky flows represent entire data processing flows. This command allows you to interact with your
shnorky flows (add new flows, inspect existing flows, remove unwanted flows from your shnorky state,
and build and execute flows).
`,
	}

	createFlowCommand := &cobra.Command{
		Use:   "create",
		Short: "Add a flow to shnorky",
		Long:  "Adds a new flow to shnorky and makes it available in the state database",
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.WithFields(
				logrus.Fields{
					"id":                id,
					"specificationPath": specificationPath,
					"stateDir":          stateDir,
				},
			)

			logger.Debug("Opening state database")
			db := openStateDB(stateDir)
			defer db.Close()

			logger.Debug("Adding component to state database")
			flow, err := flows.AddFlow(db, id, specificationPath)
			if err != nil {
				logger.WithField("error", err).Fatal("Failed to add flow")
			}
			logger.Info("Flow added successfully")

			marshalledFlow, err := json.Marshal(flow)
			if err != nil {
				logger.Fatal("Failed to marshall added flow")
			}
			fmt.Println(string(marshalledFlow))
		},
	}

	createFlowCommand.Flags().StringVarP(&id, "id", "i", "", "ID for the flow being added")

	createFlowCommand.Flags().StringVarP(&specificationPath, "spec", "s", "", "Path to flow specification")

	buildFlowCommand := &cobra.Command{
		Use:   "build",
		Short: "Build all components in a flow",
		Long:  "Creates a build for each distinct component in the given flow",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			buildsMetadata, err := flows.Build(ctx, db, dockerClient, os.Stdout, id)
			if err != nil {
				log.WithField("error", err).Fatal("Could not build components")
			}

			fmt.Println("Builds:")
			for component, buildMetadata := range buildsMetadata {
				fmt.Printf("  - %s: %s\n", component, buildMetadata.ID)
			}
		},
	}

	buildFlowCommand.Flags().StringVarP(&id, "id", "i", "", "ID for the flow to build")

	executeFlowCommand := &cobra.Command{
		Use:   "execute",
		Short: "Execute a shnorky flow",
		Long:  "Executes a shnorky flow",
		Run: func(cmd *cobra.Command, args []string) {
			db := openStateDB(stateDir)
			defer db.Close()

			dockerClient := generateDockerClient()

			ctx := context.Background()

			mounts, err := flows.ReadMountConfiguration(strings.NewReader(mountConfig))
			if err != nil {
				log.WithField("error", err).Fatal("Error reading mount configuration")
			}

			executions, err := flows.Execute(ctx, db, dockerClient, id, mounts)
			if err != nil {
				log.WithField("error", err).Fatal("Could not execute flow")
			}

			fmt.Println(executions)
		},
	}

	executeFlowCommand.Flags().StringVarP(&id, "id", "i", "", "ID of the flow being executed")
	executeFlowCommand.Flags().StringVarP(&mountConfig, "mounts", "m", "", "JSON string specifying mount configuration for flow")

	flowsCommand.AddCommand(createFlowCommand, buildFlowCommand, executeFlowCommand)

	shnorkyCommand.AddCommand(versionCommand, completionCommand, stateCommand, componentsCommand, flowsCommand)

	err = shnorkyCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
