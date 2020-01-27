package main

import (
	"bufio"
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"

	"github.com/simiotics/shnorky/components"
	"github.com/simiotics/shnorky/flows"
	"github.com/simiotics/shnorky/state"
)

func TestSingleComponent(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "shnorky-TestSingleComponent-")
	if err != nil {
		t.Fatalf("Could not create directory to hold Shnorky state: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Error initializing Shnorky state directory: %s", err.Error())
	}
	defer os.RemoveAll(stateDir)

	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	componentID := "test-component"
	componentPath := "examples/components/single-task"
	specificationPath := "examples/components/single-task/component.json"
	component, err := components.AddComponent(db, componentID, components.Task, componentPath, specificationPath)
	if err != nil {
		t.Fatalf("Error registering component: %s", err.Error())
	}

	if component.ID != componentID {
		t.Fatalf("Unexpected component ID: expected=%s, actual=%s", componentID, component.ID)
	}
	if component.ComponentType != components.Task {
		t.Fatalf("Unexpected component type: expected=%s, actual=%s", components.Task, component.ComponentType)
	}
	if component.ComponentPath != componentPath {
		t.Fatalf("Unexpected component path: expected=%s, actual=%s", componentPath, component.ComponentPath)
	}
	if component.SpecificationPath != specificationPath {
		t.Fatalf("Unexpected component path: expected=%s, actual=%s", specificationPath, component.SpecificationPath)
	}

	dockerClient := generateDockerClient()
	ctx := context.Background()

	build, err := components.CreateBuild(ctx, db, dockerClient, ioutil.Discard, component.ID)
	if err != nil {
		t.Fatalf("Error building image for component: %s", err.Error())
	}
	if build.ComponentID != component.ID {
		t.Fatalf("Unexpected component ID on build: expected=%s, actual=%s", component.ID, build.ComponentID)
	}

	imageInfo, _, err := dockerClient.ImageInspectWithRaw(ctx, build.ID)
	if err != nil {
		t.Fatalf("Could not inspect image with tag: %s", build.ID)
	}
	defer dockerClient.ImageRemove(ctx, imageInfo.ID, dockerTypes.ImageRemoveOptions{Force: true, PruneChildren: true})

	buildTags := map[string]bool{}
	for _, tag := range imageInfo.RepoTags {
		buildTags[tag] = true
	}

	if _, ok := buildTags[build.ID]; !ok {
		t.Fatalf("Expected tag (%s) was not registered against docker daemon", build.ID)
	}

	tagParts := strings.Split(build.ID, ":")
	if len(tagParts) > 1 {
		tagParts[len(tagParts)-1] = "latest"
	}
	latestTag := strings.Join(tagParts, ":")
	if _, ok := buildTags[latestTag]; !ok {
		t.Fatalf("Expected tag (%s) was not registered against docker daemon", latestTag)
	}

	// Mount configuration. The values here come from different specification files in the examples
	// directory. The values here should reflect the values there - the specification files are the
	// major source of truth. The mount paths come from examples/components/single-task/component.json
	inputFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file to mount as flow input: %s", err.Error())
	}
	inputFile.Close()
	defer os.Remove(inputFile.Name())

	outputFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file to mount as flow output: %s", err.Error())
	}
	defer os.Remove(outputFile.Name())

	mounts := []components.MountConfiguration{
		{
			Source: inputFile.Name(),
			Target: "/shnorky/inputs/inputs.txt",
			Method: "bind",
		},
		{
			Source: outputFile.Name(),
			Target: "/shnorky/outputs/outputs.txt",
			Method: "bind",
		},
	}

	execution, err := components.Execute(ctx, db, dockerClient, build.ID, "", mounts)
	if err != nil {
		t.Fatalf("Error executing build (%s): %s", build.ID, err.Error())
	}
	exitCode, err := dockerClient.ContainerWait(ctx, execution.ID)
	if err != nil {
		t.Fatalf("Error waiting for container (ID: %s) to exit: %s", execution.ID, err.Error())
	}
	if exitCode != 0 {
		t.Fatalf("Received non-zero exit code (%d) from container (ID: %s)", exitCode, execution.ID)
	}
	defer dockerClient.ContainerRemove(ctx, execution.ID, dockerTypes.ContainerRemoveOptions{})

	scanner := bufio.NewScanner(outputFile)
	defer outputFile.Close()
	more := scanner.Scan()
	if !more {
		t.Fatal("Not enough lines in output file")
	}
	line := scanner.Text()

	// expectedLine is the value for the MY_ENV variable in the component specification in:
	// examples/components/single-task/component.json
	expectedLine := "hello world"

	if line != expectedLine {
		t.Fatalf("Incorrect value in output file: expected=\"%s\", actual=\"%s\"", expectedLine, line)
	}

	terminating := 0
	for scanner.Scan() {
		terminating++
		line = scanner.Text()
		if line != "" {
			t.Fatalf("Got unexpected non-empty line from output file: %s", line)
		}
	}

	if terminating > 1 {
		t.Fatalf("Too many terminating newlines in output file: %d", terminating)
	}

	// TODO(nkashy1): Implement execution state management and add those functions into this test
}

func TestFlowSingleTaskTwice(t *testing.T) {
	stateDir, err := ioutil.TempDir("", "shnorky-TestFlowSingleTaskTwice-")
	if err != nil {
		t.Fatalf("Could not create directory to hold Shnorky state: %s", err.Error())
	}
	os.RemoveAll(stateDir)

	err = state.Init(stateDir)
	if err != nil {
		t.Fatalf("Error initializing Shnorky state directory: %s", err.Error())
	}
	defer os.RemoveAll(stateDir)

	stateDBPath := path.Join(stateDir, state.DBFileName)
	db, err := sql.Open("sqlite3", stateDBPath)
	if err != nil {
		t.Fatal("Error opening state database file")
	}
	defer db.Close()

	componentID := "single-task"
	componentPath := "examples/components/single-task"
	specificationPath := "examples/components/single-task/component.json"
	component, err := components.AddComponent(db, componentID, components.Task, componentPath, specificationPath)
	if err != nil {
		t.Fatalf("Error registering component: %s", err.Error())
	}

	if component.ID != componentID {
		t.Fatalf("Unexpected component ID: expected=%s, actual=%s", componentID, component.ID)
	}
	if component.ComponentType != components.Task {
		t.Fatalf("Unexpected component type: expected=%s, actual=%s", components.Task, component.ComponentType)
	}
	if component.ComponentPath != componentPath {
		t.Fatalf("Unexpected component path: expected=%s, actual=%s", componentPath, component.ComponentPath)
	}
	if component.SpecificationPath != specificationPath {
		t.Fatalf("Unexpected component path: expected=%s, actual=%s", specificationPath, component.SpecificationPath)
	}

	flowID := "flow-single-task-twice"
	flowSpecificationPath := "examples/flows/single-task-twice.json"
	flow, err := flows.AddFlow(db, flowID, flowSpecificationPath)
	if err != nil {
		t.Fatalf("Error registering flow: %s", err.Error())
	}

	if flow.ID != flowID {
		t.Fatalf("Unexpected flow ID: expected=%s, actual=%s", flowID, flow.ID)
	}
	if flow.SpecificationPath != flowSpecificationPath {
		t.Fatalf("Unexpected flow ID: expected=%s, actual=%s", flowSpecificationPath, flow.SpecificationPath)
	}

	dockerClient := generateDockerClient()
	ctx := context.Background()

	flowBuilds, err := flows.Build(ctx, db, dockerClient, ioutil.Discard, flow.ID)
	if err != nil {
		t.Fatalf("Error building images for flow: %s", err.Error())
	}

	for flowComponent, flowBuild := range flowBuilds {
		if flowBuild.ComponentID != flowComponent {
			t.Fatalf("Unexpected component ID on build: expected=%s, actual=%s", flowComponent, flowBuild.ComponentID)
		}

		imageInfo, _, err := dockerClient.ImageInspectWithRaw(ctx, flowBuild.ID)
		if err != nil {
			t.Fatalf("Could not inspect image with tag: %s", flowBuild.ID)
		}
		defer dockerClient.ImageRemove(ctx, imageInfo.ID, dockerTypes.ImageRemoveOptions{Force: true, PruneChildren: true})

		buildTags := map[string]bool{}
		for _, tag := range imageInfo.RepoTags {
			buildTags[tag] = true
		}

		if _, ok := buildTags[flowBuild.ID]; !ok {
			t.Fatalf("Expected tag (%s) was not registered against docker daemon", flowBuild.ID)
		}

		tagParts := strings.Split(flowBuild.ID, ":")
		if len(tagParts) > 1 {
			tagParts[len(tagParts)-1] = "latest"
		}
		latestTag := strings.Join(tagParts, ":")
		if _, ok := buildTags[latestTag]; !ok {
			t.Fatalf("Expected tag (%s) was not registered against docker daemon", latestTag)
		}

	}

	// Mount configuration. The values here come from different specification files in the examples
	// directory. The values here should reflect the values there - the specification files are the
	// major source of truth:
	// 1. Step names come from examples/flows/single-task-twice.json
	// 2. Component mount paths come from examples/components/single-task/component.json
	inputFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file to mount as flow input: %s", err.Error())
	}
	inputFile.Close()
	defer os.Remove(inputFile.Name())

	intermediateFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file to mount as flow connector: %s", err.Error())
	}
	intermediateFile.Close()
	defer os.Remove(intermediateFile.Name())

	outputFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file to mount as flow output: %s", err.Error())
	}
	defer os.Remove(outputFile.Name())

	stepMounts := map[string][]components.MountConfiguration{
		"first": {
			{
				Source: inputFile.Name(),
				Target: "/shnorky/inputs/inputs.txt",
				Method: "bind",
			},
			{
				Source: intermediateFile.Name(),
				Target: "/shnorky/outputs/outputs.txt",
				Method: "bind",
			},
		},
		"second": {
			{
				Source: intermediateFile.Name(),
				Target: "/shnorky/inputs/inputs.txt",
				Method: "bind",
			},
			{
				Source: outputFile.Name(),
				Target: "/shnorky/outputs/outputs.txt",
				Method: "bind",
			},
		},
	}

	flowExecutions, err := flows.Execute(ctx, db, dockerClient, flow.ID, stepMounts)
	for _, stepExecution := range flowExecutions {
		defer dockerClient.ContainerRemove(ctx, stepExecution.ID, dockerTypes.ContainerRemoveOptions{})
	}
	if err != nil {
		t.Fatalf("Error in flow execution: %s", err.Error())
	}

	// expectedLine is the value for the MY_ENV variable in the component specification in:
	// examples/components/single-task/component.json
	expectedLine := "hello world"
	scanner := bufio.NewScanner(outputFile)
	defer outputFile.Close()
	more := scanner.Scan()
	if !more {
		t.Fatal("Not enough lines in output file")
	}
	line := scanner.Text()
	if line != expectedLine {
		t.Fatalf("Incorrect value in output file: expected=\"%s\", actual=\"%s\"", expectedLine, line)
	}

	more = scanner.Scan()
	if !more {
		t.Fatal("Not enough lines in output file")
	}
	line = scanner.Text()
	if line != expectedLine {
		t.Fatalf("Incorrect value in output file: expected=\"%s\", actual=\"%s\"", expectedLine, line)
	}

	terminating := 0
	for scanner.Scan() {
		terminating++
		line = scanner.Text()
		if line != "" {
			t.Fatalf("Got unexpected non-empty line from output file: %s", line)
		}
	}

	if terminating > 1 {
		t.Fatalf("Too many terminating newlines in output file: %d", terminating)
	}
}
