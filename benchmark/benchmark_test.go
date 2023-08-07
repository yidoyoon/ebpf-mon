package benchmark

import (
	"encoding/json"
	"fmt"
	v2 "github.com/yidoyoon/cadvisor-lite/info/v2"
	bf "github.com/yidoyoon/cadvisor-lite/integration/benchframework"
	"github.com/yidoyoon/cadvisor-lite/manager"
	"log"
	"os/signal"
	"syscall"

	"github.com/yidoyoon/ebpf-mon/helper"
	"k8s.io/klog/v2"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func setup(b *testing.B) (manager.Manager, error) {
	if !helper.IsDockerInstalled() {
		b.Fatal("Docker is not installed or Docker daemon is not running.")
	}
	countString := os.Getenv("COUNT")
	currentContainerCount := len(helper.GetAllContainers())
	if countString == "" {
		b.Logf("COUNT is not set.\nRun without creating new containers...\nCurrent containers: %d", currentContainerCount)
	} else {
		count, err := strconv.Atoi(countString)
		if err != nil || count <= 0 {
			b.Fatal(err)
		}
		helper.CreateTestContainer(count)
	}

	return CreateManager()
}

func teardown() {
	fmt.Println("\nTest is done. Clean up dummy containers...")
	_, err := exec.Command("sh", "-c", "docker rm -f $(docker ps -a -q --filter=name=busybox-ebpfmon)").Output()
	if err != nil {
		fmt.Println("There are no dummy containers to remove.")
	}
}

func getMemoryUsageWithProc() string {
	data, err := os.ReadFile("/proc/metric")
	if err != nil {
		log.Fatal(err)
	}

	return string(data)
}

func writeResult(res interface{}) error {
	_, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("failed to marshall response %+v with error: %s", res, err)
	}

	return err
}

func getMemoryUsageWithCadvisorScrapeOnly(m manager.Manager, opt *v2.RequestOptions) error {
	name := "/"

	klog.V(4).Infof("Api - Stats: Looking for stats for container %q, options %+v", name, opt)
	infos, err := m.GetRequestedContainersInfo(name, *opt)
	if err != nil {
		klog.Errorf("Error calling GetRequestedContainersInfo: %v", err)
	}
	contStats := make(map[string][]v2.DeprecatedContainerStats)
	for name, cinfo := range infos {
		contStats[name] = v2.DeprecatedStatsFromV1(cinfo)
	}

	return writeResult(contStats)
}

func installSignalHandler(containerManager manager.Manager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		if err := containerManager.Stop(); err != nil {
			klog.Errorf("Failed to stop container manager: %v", err)
		}
		klog.Infof("Exiting given signal: %v", sig)
		os.Exit(0)
	}()
}

func Benchmark(b *testing.B) {
	m, err := setup(b)
	if err != nil {
		b.Fatal("Cannot create manager.")
	}
	if err := m.Start(); err != nil {
		klog.Fatalf("Failed to start manager: %v", err)
	}

	defer b.Cleanup(teardown)

	containers := helper.GetAllContainers()
	containerCount := len(containers)
	if containerCount == 0 {
		b.Fatal("No containers found. Exit benchmark...")
	}

	bm := bf.New(b)
	requestOptions := &v2.RequestOptions{
		Count:     1,
		Recursive: true,
		IdType:    v2.TypeName,
	}

	b.Run(fmt.Sprintf("kernelModule"), func(b *testing.B) {
		for j := 0; j < b.N; j++ {
			getMemoryUsageWithProc()
		}
	})

	b.Run(fmt.Sprintf("cadvisorClient"), func(b *testing.B) {
		for j := 0; j < b.N; j++ {
			_, err := bm.Cadvisor().ClientV2().Stats("", requestOptions)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	//installSignalHandler(m)
	//
	//b.Run(fmt.Sprintf("cadvisorScrapeOnly"), func(b *testing.B) {
	//	for j := 0; j < b.N; j++ {
	//		err := getMemoryUsageWithCadvisorScrapeOnly(m, requestOptions)
	//		if err != nil {
	//			b.Fatal(err)
	//		}
	//	}
	//})

}
