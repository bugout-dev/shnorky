# shnorky

Shnorky is a workflow orchestrator for data scientists. Shnorky allows you to build and manage your data flows.

As a data scientist, you can use Shnorky to string together your notebooks and scripts into a
complete pipeline from data processing to model training, evaluation, and deployment.

Your Shnorky pipelines will run the same way in production as they do on your laptop or workstation.
This means that you can get your work into production quickly and without any back-and-forths with a
deployment team.

### Advantages of using Shnorky

- Agency for data scientists: You don’t need DevOps support.
- Easy to deploy: Write your data flows locally and export them directly into production.
- Automation: Build  your pipelines much faster with pre-built steps.

### Testimonials from Shnorky users

*"With Shnorky, I was able to easily containerize and deploy my software to the cloud. Every data scientist must have it in their tech stack."*

-- Art Ponomarev, Senior Data Scientist

To try it out, follow the instructions below.

## Requirements

+ [`docker`](https://docs.docker.com/install/) - Shnorky uses docker to run workflow components.

+ [`go`](https://golang.org/) - Shnorky is written in Go. We are working on releasing pre-built
binaries for your environment. Until then, the easiest way to install Shnorky is using Go's `go get`
command.

+ [`gcc`](https://gcc.gnu.org/) (use [`mingw`](http://www.mingw.org/) on Windows)

## Installation

```
go get github.com/simiotics/shnorky/cmd/shn
```

This will put the Shnorky `shn` binary in your `$(go env GOPATH)/bin` directory (most likely at
`~/go/bin`).

## Usage

Shnorky builds and executes *flows* which represent entire data science pipelines. A flow consists
of multiple *components* and encodes the dependencies between these components.

Any program can be a component. The only requirement is that it be runnable in a Docker container.
You register components in Shnorky by providing a directory containing the program (complete with
instructions for building a Docker image for that component) and a specification file which
describes, among other things, the inputs a component expects and the outputs it produces.

For example, [`examples/components/single-task/component.json`](examples/components/single-task/component.json)
is a component which reads the file at `/shnorky/inputs.txt` and writes its output to `/shnorky/outputs.txt`.
The code it executes is provided in the [Dockerfile](examples/components/single-task/Dockerfile) - it
appends a new line containing either the string `+1` or the value of the `MY_ENV` environment variable
to its input file.

[`examples/flows/single-task-twice.json`](examples/flows/single-task-twice.json) is a complete definition
of a Shnorky flow which runs the component defined above twice, in a series. This is where the inputs
and outputs in the component containers get connected to files on the host running the flow.

To run the example flow on your machine, follow these instructions:

### Initialize state database

Shnorky stores information about which flows and components are registered against it and when they
were run in a state directory. Before you can use Shnorky, you must ask it to prepare this state
directory:

```
shn state init
```

### Register a component

Flows refer to pre-registered Shnorky components. So let us start by registering the line appender
component:

```
shn components create -c examples/components/single-task -i single-task -t task
```

### Register a flow

Now we can register the example flow:

```
shn flows create -i single-task-twice -s examples/flows/single-task-twice.json
```

### Build images for all components in a flow

Before we can run a flow, we must build Docker images for each component in the flow:

```
shn flows build -i single-task-twice
```

### Execute a flow

The sample flow requires three files to exist (`inputs.txt`, `intermediate.txt`, and `outputs.txt`).
Create these files:
```
touch inputs.txt intermediate.txt outputs.txt
```

Set the following environment variables (use `set` on Windows):
```
export SHNORKY_TEST_INPUT=inputs.txt SHNORKY_TEST_INTERMEDIATE=intermediate.txt SHNORKY_TEST_OUTPUT=outputs.txt
```

Then, from the same shell, run the flow:
```
shn flows execute -i single-task-twice
```

## Help

For help, [create a GitHub issue in this repository](https://github.com/simiotics/shnorky/issues/new).
Alternatively, email engineering@simiotics.com
