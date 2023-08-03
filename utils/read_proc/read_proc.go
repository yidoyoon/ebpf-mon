package main

import (
	"fmt"
	"github.com/hodgesds/perf-utils"
	"io"
	"log"
	"os/exec"
)

func main() {
	cmd := exec.Command("cat", "/proc/metric")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}

	getStats := func() error {
		cmd := exec.Command("cat", "/proc/metric")
		_, err := cmd.Output()
		return err
	}

	cpuInstructions, _ := perf.CPUInstructions(getStats)
	cpuCycles, _ := perf.CPUCycles(getStats)
	cacheRef, _ := perf.CacheRef(getStats)
	cacheMiss, _ := perf.CacheMiss(getStats)
	cpuRefCycles, _ := perf.CPURefCycles(getStats)
	cpuClock, _ := perf.CPUClock(getStats)
	cpuTaskClock, _ := perf.CPUTaskClock(getStats)
	pageFaults, _ := perf.PageFaults(getStats)
	contextSwitches, _ := perf.ContextSwitches(getStats)
	minorPageFaults, _ := perf.MinorPageFaults(getStats)
	majorPageFaults, _ := perf.MajorPageFaults(getStats)

	fmt.Println(cpuInstructions.Value, cpuCycles.Value, cacheRef.Value, cacheMiss.Value, cpuRefCycles.Value, cpuClock.Value, cpuTaskClock.Value, pageFaults.Value, contextSwitches.Value, minorPageFaults.Value, majorPageFaults.Value)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(io.Discard, string(output))
}
