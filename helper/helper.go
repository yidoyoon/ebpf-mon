package helper

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

func GetAllContainers() []types.Container {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})

	return containers
}

func IsDockerInstalled() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	_, err = cli.ServerVersion(ctx)
	if err != nil {
		return false
	}

	return true
}

func CreateTestContainer(count int) {
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			containerName := "busybox-ebpfmon" + strconv.Itoa(i)
			_, err := exec.Command("sh", "-c", "docker run -itd --rm --name "+containerName+" busybox").Output()
			if err != nil {
				fmt.Printf("Failed to create container: %v\n", err)
			} else {
				fmt.Printf("Successfully created container: %s\n", containerName)
			}
		}(i)
	}
	wg.Wait()

	fmt.Println("Wait for containers are initialized...")
	time.Sleep(5 * time.Second)
}

func InputCount() int {
	fmt.Println("Enter the number of docker containers(busybox) to create: ")
	var containerCount int
	_, err := fmt.Scan(&containerCount)
	if err != nil {
		fmt.Println(err)
	}

	return containerCount
}
