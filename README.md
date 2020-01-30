# shnorky

Shnorky is a workflow orchestrator which:

1. Runs on a single machine

2. Composes workflows using docker

3. Targets data processing flows


## Requirements

+ [`docker`](https://docs.docker.com/install/) - Shnorky uses docker to run workflow components

## Installation

### go get

#### Requirements

+ [go](https://golang.org/) - go.1.13.0 or greater

#### Steps

If you have `go` installed on your computer, you can get Shnorky using `go get`:
```
go get github.com/simiotics/shnorky
```

This will put the Shnorky `shn` binary in your `\`go env GOPATH\`/bin` directory.

### From source

#### Requirements

+ [go](https://golang.org/) - go.1.13.0 or greater

+ [gcc](https://gcc.gnu.org/) - known to work with gcc 5.4.0 and greater

+ [GNU Make](https://www.gnu.org/software/make/) - known to work with make 4.1

#### Steps

Clone this repository:
```
git clone https://github.com/simiotics/shnorky.git
```

Move into the cloned directory:
```
cd shnorky
```

Make the `shn` binary:
```
make build
```

This will create a `shn` binary in that directory, which you can test:
```
./shn -h
```

To make this binary available globally, run:
```
sudo mv ./shn /usr/local/bin/
```

Test again:
```
shn -h
```

## Usage

These usage examples use the example flows and components in the [`examples`](./examples) directory.

### Initialize state database

First, determine where you would like to put the Shnorky state directory - which should not already
exist before you run the initialization command. Then:

```
shn -S <PATH TO STATE DIRECTORY> state init
```

### Register a component

```
shn -S <PATH TO STATE DIRECTORY> components create -c examples/components/single-task -i single-task -t task
```

### Register a flow

```
shn -S <PATH TO STATE DIRECTORY> flows create -i single-task-twice -s examples/flows/single-task-twice.json
```

### Build images for all components in a flow

```
shn -S <PATH TO STATE DIRECTORY> flows build -i single-task-twice
```

### Execute a flow

The sample flow requires three files to exist (`inputs.txt`, `intermediate.txt`, and `outputs.txt`).
Create these files:
```
touch inputs.txt intermediate.txt outputs.txt
```

Then, run the flow:
```
shn -S <PATH TO STATE DIRECTORY> flows execute \
    -m "{\"first\": [{\"source\": \"$PWD/inputs.txt\", \"target\": \"/shnorky/inputs/inputs.txt\", \"method\": \"bind\"}, {\"source\": \"$PWD/intermediate.txt\", \"target\": \"/shnorky/outputs/outputs.txt\", \"method\": \"bind\"}], \"second\": [{\"source\": \"$PWD/intermediate.txt\", \"target\": \"/shnorky/inputs/inputs.txt\", \"method\": \"bind\"}, {\"source\": \"$PWD/outputs.txt\", \"target\": \"/shnorky/outputs/outputs.txt\", \"method\": \"bind\"}]}" \
    -i single-task-twice
```

## Rationale

Data science begins with data processing. Data processing, in the absence of scale, is not Cool. It
is often performed using A Bunch of Scripts (TM) which may or may not be version-controlled or even
available on a single machine.

If you need to process large amounts of data, there are many tools available to help you do so. Many
of them start with the prefix `"Apache "` (for example, [Airflow](https://airflow.apache.org/) and
[Spark](https://spark.apache.org/)). Such tools encourage you to bring up clusters of machines to
run your data processing flows in production environments. For teams that do not operate at the
scale these tools are designed for, these ceremonies introduce unnecessary overhead - often taking
non-trivial amounts of maintenance effort every week.

Shnorky makes strong but simplifying assumptions about the environment in which it will run:

1. It will run all components of a flow on a single machine.

2. It will run each component of a flow in a Docker container.

3. It is sufficient to store the metadata related to its flows, their components, and each execution
in a local database.

Wherever possible, Shnorky encourages use of the file system for communication between components in
a workflow. This saves you from having to set up (and maintain) a RabbitMQ or Redis cluster.

Shnorky stores all metadata in a SQLite database on the same machine running the workflows. This
saves you from having to set up (and maintain) a separate database server for Shnorky metadata.

All this mean that there is no difference to Shnorky between a production and a development
environment. Generally all you have to do to run a workflow in production is develop it locally,
commit it to a git repo of your choice, clone that repo in your production environment (best done
with CI tools), register the flow (also using CI tools), and schedule it (using CI or manually, we
like [`cron`](https://en.wikipedia.org/wiki/Cron) for this).

If you are already using A Bunch of Scripts (TM) to implement your data processing flows, it is easy
to run them using Shnorky. Our [`examples/`](./examples) directory has samples you can copy from.

Shnorky is inspired by [`docker-compose`](https://github.com/docker/compose). It extends the
functionality of `docker-compose` to cover dependencies for data processing tasks.

## Help

For help, email engineering@simiotics.com
