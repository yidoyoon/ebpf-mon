package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type PidName struct {
	pid  string
	name string
}

var containerListFile = ".container_info"

const MaxRetries = 9

func SetupSignalHandling() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGSEGV)
	go func() {
		sig := <-c
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGSEGV:
			if _, err := exec.Command("sh", "-c", "rmmod metric.ko").Output(); err != nil {
				fmt.Printf("Failed to remove module: %v\n", err)
			}
			if _, err := exec.Command("sh", "-c", "docker rm -f $(docker ps -a -q --filter=name=busybox-ebpfmon)").Output(); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			log.Fatalf("%s received, stopping module and program will exit.\n", sig)
		case syscall.SIGILL, syscall.SIGFPE, syscall.SIGPIPE:
			log.Fatalf("An internal error occurred: %s, program will exit.\n", sig)
		default:
			log.Printf("Please stop the program using Ctrl+C.\n")
		}
	}()
}

func timePrint(msg string) {
	now := time.Now()
	fmt.Println(now.Format("01-02 15:04:05") + "\t" + msg)
}

func InsertModule(pidName []PidName) {
	pidLen := len(pidName)
	if pidLen <= 0 {
		fmt.Println("There are no running containers in the current namespace.")
	}

	var pids string
	var names string

	for _, pair := range pidName[:pidLen] {
		pids += pair.pid + string(',')
		names += pair.name + string(',')
	}

	cmd := "insmod metric.ko pid=" + pids[:len(pids)-1] + " pid_count=" + fmt.Sprint(pidLen) + " container_name=" + names[:len(names)-1]
	fmt.Println(cmd)
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Fatalf("Cannot find metric.ko\nPlease check the module has loaded correctly.")
	}

	timePrint("[INFO] Container count(s) : " + fmt.Sprint(pidLen))
}

func RemoveModule() {
	exec.Command("sh", "-c", "rmmod metric.ko").Output()
}

func ContainerChangeDetector(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		var pidName []PidName
		currentContainerLists := GetCurrentContainerLists()
		prevHash := GetPrevHash(containerListFile)
		currentHash := GetCurrentHash(currentContainerLists)
		if prevHash != currentHash {
			trimmedData := strings.TrimSpace(string(currentContainerLists))
			containerList := strings.Split(trimmedData, "\n")
			for _, container := range containerList {
				slice := strings.Split(container, ",")
				pidName = append(pidName, PidName{slice[0], slice[2][1:]})
			}
			RemoveModule()
			InsertModule(pidName)
			err := os.WriteFile(containerListFile, currentContainerLists, 0644)
			if err != nil {
				log.Fatalf("Error occurred while writing to file: %v", err)
			}

			timePrint("[INFO] Container list has been updated")
		}
	}
}

func ResetContainerInfo() {
	err := os.WriteFile(containerListFile, []byte{}, 0644)
	if err != nil {
		log.Fatalf("Error occurred while writing to file: %v", err)
	}
}

func GetCurrentContainerLists() []byte {
	retryCount := 0
	for {
		currentContainerData, err := exec.Command("sh", "-c", "docker ps -q|xargs docker inspect --format '{{.State.Pid}},{{.ID}},{{.Name}}'").Output()
		if err == nil {
			return currentContainerData
		}

		retryCount++
		fmt.Printf("No containers currently running. Retrying(%d)...\n", retryCount)
		if retryCount >= MaxRetries {
			log.Fatalf("Error fetching container data for %d consecutive attempts. Exit...", MaxRetries)
		}

		time.Sleep(10 * time.Second)
	}
}

func GetPrevHash(containerListFile string) string {
	prevContainerData, _ := os.ReadFile(containerListFile)
	return GetHash(string(prevContainerData))
}

func GetCurrentHash(currentContainerLists []byte) string {
	return GetHash(string(currentContainerLists))
}

func GetHash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func CreateTestContainer() {
	fmt.Println("How many containers(busybox)?: ")
	var count int
	fmt.Scanf("%d", &count)
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
}

func main() {
	ResetContainerInfo()
	SetupSignalHandling()

	var wg sync.WaitGroup
	wg.Add(1)

	if os.Getuid() != 0 {
		fmt.Println("This program must be run as root. Please run again with root privileges.")
		os.Exit(1)
	}

	fmt.Println("Do you want to create some dummy containers? (y/N)")
	var command string
	_, err := fmt.Scanln(&command)
	for err != nil || (command != "Y" && command != "y" && command != "N" && command != "n") {
		if err != nil {
			fmt.Println("An error occurred:", err)
		} else {
			fmt.Println("The input is not valid. Please type 'y' for yes, 'N' for no.")
		}
		_, err = fmt.Scanln(&command)
	}
	if command == "Y" || command == "y" {
		CreateTestContainer()
	}

	go ContainerChangeDetector(&wg)
	wg.Wait()
}
