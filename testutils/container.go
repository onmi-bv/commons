package testutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// CreateNewContainer creates a new container, and binding to the hostPort.
func CreateNewContainer(image string, hostPort string, containerPort string, env []string) (*client.Client, string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		fmt.Println("Unable to create docker client")
		panic(err)
	}

	// // first pull the image in case it doesnt exist
	// ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	// defer cancel()
	// if _, err := cli.ImagePull(ctx, image, types.ImagePullOptions{}); err != nil {
	// 	fmt.Println("Unable to pull docker image")
	// 	panic(err)
	// }

	// create host port binding
	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: hostPort,
	}

	// create container nat port
	containerPortNat, err := nat.NewPort("tcp", containerPort)
	if err != nil {
		panic("Unable to get the port")
	}

	// Bind container to host port binding
	portBinding := nat.PortMap{containerPortNat: []nat.PortBinding{hostBinding}}

	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: image,
			Env:   env,
		},
		&container.HostConfig{
			PortBindings: portBinding,
		}, nil, "")
	if err != nil {
		panic(err)
	}

	cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	fmt.Printf("Container %s %s is started\n", image, cont.ID)
	return cli, cont.ID, nil
}

// RemoveContainer removes a running container by its ID
func RemoveContainer(cID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, _ := client.NewEnvClient()

	err := client.ContainerRemove(ctx, cID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		log.Fatalf("cannot remove container: %v", err)
	}

	client.ContainerWait(ctx, cID)
}
