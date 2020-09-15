package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"log"
	"os"
)

var (
	// ServerSession is the HTTP session with the identity server
	quit   chan os.Signal
	logger *log.Logger
	stop   chan bool
)

func main() {
	logger = log.New(os.Stdout, "bootstrap-controller: ", log.LstdFlags)
	logger.Println("Trying to create a simple nginx container")

	// Create a new docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Println(err)
		return
	}

	// Get containers list
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		logger.Println(err)
		return
	}

	isContainerFound := false

	match := "/nginx-test"
	var c types.Container

	for _, container := range containers {
		if container.Names[0] == match {
			isContainerFound = true
			c = container
			break
		}
	}

	if isContainerFound {
		logger.Println("container already started: checking the exposed port")

		inspect, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			logger.Fatal(err)
		}

		ports := inspect.NetworkSettings.Ports
		logger.Println("looking for the host port for 80/tcp in container")
		for k, p := range ports {
			if k.Port() != "80" {
				continue
			}
			if k.Proto() != "tcp" {
				continue
			}
			logger.Println("found mapped port 80/tcp -> " + p[0].HostPort + "/tcp")
		}

		return
	}

	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs([]string{"80/tcp"})
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        "nginx",
		ExposedPorts: exposedPortSet,
	}, &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		PortBindings: exposedPortMap,
	}, nil, "nginx-test")

	if err != nil {
		logger.Println(err)
		return
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logger.Println(err)
		return
	}

	fmt.Printf("container 'nginx-test' started with ID: %v", resp.ID)
}
