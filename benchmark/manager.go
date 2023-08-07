package benchmark

import (
	"crypto/tls"
	"github.com/yidoyoon/cadvisor-lite/container"
	"github.com/yidoyoon/cadvisor-lite/manager"
	"github.com/yidoyoon/cadvisor-lite/utils/sysfs"
	"strings"

	"k8s.io/klog/v2"
	"net/http"
)

func CreateManager() (manager.Manager, error) {
	memoryStorage, err := NewMemoryStorage()
	if err != nil {
		klog.Fatalf("Failed to initialize storage driver: %s", err)
	}
	ignoreMetrics := container.MetricSet{
		container.MemoryNumaMetrics:              struct{}{},
		container.NetworkTcpUsageMetrics:         struct{}{},
		container.NetworkUdpUsageMetrics:         struct{}{},
		container.NetworkAdvancedTcpUsageMetrics: struct{}{},
		container.ProcessSchedulerMetrics:        struct{}{},
		container.ProcessMetrics:                 struct{}{},
		container.HugetlbUsageMetrics:            struct{}{},
		container.ReferencedMemoryMetrics:        struct{}{},
		container.CPUTopologyMetrics:             struct{}{},
		container.ResctrlMetrics:                 struct{}{},
		container.CPUSetMetrics:                  struct{}{},
	}

	sysFs := sysfs.NewRealSysFs()
	enableMetrics := container.MetricSet{
		container.MemoryUsageMetrics: struct{}{},
	}
	var includedMetrics container.MetricSet
	if len(enableMetrics) > 0 {
		includedMetrics = enableMetrics
	} else {
		includedMetrics = container.AllMetrics.Difference(ignoreMetrics)
	}

	collectorHTTPClient := createCollectorHTTPClient("", "")

	resourceManager, err := manager.New(memoryStorage, sysFs, manager.HousekeepingConfigFlags, includedMetrics, &collectorHTTPClient, strings.Split("", ","), strings.Split("", ","), "", 0)

	return resourceManager, err
}

func createCollectorHTTPClient(collectorCert, collectorKey string) http.Client {
	//Enable accessing insecure endpoints. We should be able to access metrics from any endpoint
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if collectorCert != "" {
		if collectorKey == "" {
			klog.Fatal("The collector_key value must be specified if the collector_cert value is set.")
		}
		cert, err := tls.LoadX509KeyPair(collectorCert, collectorKey)
		if err != nil {
			klog.Fatalf("Failed to use the collector certificate and key: %s", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate() //nolint: staticcheck
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return http.Client{Transport: transport}
}
