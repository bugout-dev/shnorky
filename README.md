# simplex

Simplex is a workflow orchestrator which:

1. Runs on a single machine

2. Composes workflows using docker

3. Targets data processing flows


## Requirements

+ [`docker`](https://docs.docker.com/install/) - Simplex uses docker to run workflow components

## Installation

### From source

#### Requirements

+ [go](https://golang.org/) - go.1.13.0 or greater

+ [gcc](https://gcc.gnu.org/) - known to work with gcc 5.4.0 and greater

+ [GNU Make](https://www.gnu.org/software/make/) - known to work with make 4.1

#### Steps

Clone this repository:
```
git clone https://github.com/simiotics/simplex.git
```

Move into the cloned directory:
```
cd simplex
```

Make the `simplex` binary:
```
make build
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

Simplex makes strong but simplifying assumptions about the environment in which it will run:

1. It will run all components of a flow on a single machine.

2. It will run each component of a flow in a Docker container.

3. It is sufficient to store the metadata related to its flows, their components, and each execution
in a local database.

Wherever possible, Simplex encourages use of the file system for communication between components in
a workflow. This saves you from having to set up (and maintain) a RabbitMQ or Redis cluster.

Simplex stores all metadata in a SQLite database on the same machine running the workflows. This
saves you from having to set up (and maintain) a separate database server for Simplex metadata.

All this mean that there is no difference to Simplex between a production and a development
environment. Generally all you have to do to run a workflow in production is develop it locally,
commit it to a git repo of your choice, clone that repo in your production environment (best done
with CI tools), register the flow (also using CI tools), and schedule it (using CI or manually, we
like [`cron`](https://en.wikipedia.org/wiki/Cron) for this).

If you are already using A Bunch of Scripts (TM) to implement your data processing flows, it is easy
to run them using Simplex. Our [`examples/`](./examples) directory has samples you can copy from.

Simplex is inspired by [`docker-compose`](https://github.com/docker/compose). It extends the
functionality of `docker-compose` to cover dependencies for data processing tasks.
