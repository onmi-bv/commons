package testutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ContainerConfig defines the config for creating new container
type ContainerConfig struct {
	Image   string // Container image
	PortMap []struct {
		Host      string
		Container string
	} // Maps the host port to the container port
	VolumeMap []struct {
		Source string
		Target string
	} // Maps the volume source path to the target path
	env []string // Set environment vars
}

// CreateNewContainer creates a new container, and binding to the hostPort.
func CreateNewContainer(ctx context.Context, config ContainerConfig) (*client.Client, string, error) {
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

	var portBinding = nat.PortMap{}

	for _, m := range config.PortMap {
		// create host port binding
		hostBinding := nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: m.Host,
		}

		// create container nat port
		containerPortNat, err := nat.NewPort("tcp", m.Container)
		if err != nil {
			panic("Unable to get the port")
		}

		// Bind container to host port binding
		portBinding[containerPortNat] = []nat.PortBinding{hostBinding}
	}

	var volumeMounts = []mount.Mount{}

	for _, m := range config.VolumeMap {
		volumeMounts = append(volumeMounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: m.Source,
			Target: m.Target,
		})
	}

	cont, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: config.Image,
			Env:   config.env,
		},
		&container.HostConfig{
			PortBindings: portBinding,
			Mounts:       volumeMounts,
		}, nil, "")
	if err != nil {
		panic(err)
	}

	cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	fmt.Printf("Container %s %s is started\n", config.Image, cont.ID)
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
