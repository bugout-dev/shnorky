package internal

import (
	"context"

	docker "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// GenerateDockerClient returns a docker client configured to talk to the API specified by the
// environment of the executing process
func GenerateDockerClient(log *logrus.Logger) *docker.Client {
	client, err := docker.NewEnvClient()
	if err != nil {
		log.WithField("error", err).Fatal("Error creating docker client")
	}

	ctx := context.Background()
	client.NegotiateAPIVersion(ctx)

	return client
}
