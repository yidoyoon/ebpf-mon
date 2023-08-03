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

var (
	beforeHash string
	newHash    string
)

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
			fmt.Printf("%s received, stopping module and program will exit.\n", sig)
			os.Exit(0)
		case syscall.SIGILL, syscall.SIGFPE, syscall.SIGPIPE:
			fmt.Printf("An internal error occurred: %s, program will exit.\n", sig)
			os.Exit(1)
		default:
			fmt.Printf("Please stop the program using Ctrl+C.\n")
		}
	}()
}

func timePrint(msg string) {
	now := time.Now()
	fmt.Println(now.Format("01-02 15:04:05") + "\t" + msg)
}

func InsertModule(pidName []PidName) {
	pidLen := len(pidName) - 1
	if pidLen == 0 {
		fmt.Println("There are no running containers in the current namespace.\nRemoving the module.")
		_, err := exec.Command("sh", "-c", "rmmod metric.ko").Output()
		if err != nil {
			log.Println(err)
		}
		return
	}

	var pids string
	var names string

	for _, pair := range pidName[:pidLen] {
		pids += pair.pid + string(',')
		names += pair.name + string(',')
	}

	cmd := "insmod metric.ko pid=" + pids[:len(pids)-1] + " pid_count=" + fmt.Sprint(pidLen) + " container_name=" + names[:len(names)-1]
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Println(err)
	}
	fmt.Println(cmd)

	timePrint("[INFO] Container count(s) : " + fmt.Sprint(pidLen+1))
}

func ContainerChangeDetector(wg *sync.WaitGroup) {
	defer wg.Done()
	containerData := "./.container_info"
	var pidName []PidName

	for {
		exec.Command("sh", "-c", "docker ps -q|xargs docker inspect --format '{{.State.Pid}},{{.ID}},{{.Name}}' > "+containerData).Output()
		data, _ := os.ReadFile(containerData)
		newHash = GetHash(string(data))
		if beforeHash != newHash {
			trimmedData := strings.TrimSpace(string(data))
			containerList := strings.Split(trimmedData, "\n")
			for _, container := range containerList {
				slice := strings.Split(container, ",")
				pidName = append(pidName, PidName{slice[0], slice[2][1:]})
			}
			InsertModule(pidName)

			beforeHash = newHash
			timePrint("[INFO] Container list has been updated")
			pidName = pidName[:0]
		}
	}
}

func GetHash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func InitWatcher() {
	var pidName []PidName
	containerData := "./.container_info"

	if _, err := os.Stat(containerData); os.IsNotExist(err) {
		_, err := os.Create(containerData)
		if err != nil {
			log.Fatalf("Failed creating file: %s", err)
		}
	}
	exec.Command("sh", "-c", "docker ps -q|xargs docker inspect --format '{{.State.Pid}},{{.ID}},{{.Name}}' > "+containerData).Output()
	data, _ := os.ReadFile(containerData)
	trimmedData := strings.TrimSpace(string(data))
	if trimmedData != "" {
		containerList := strings.Split(trimmedData, "\n")
		for _, container := range containerList {
			slice := strings.Split(container, ",")
			if len(slice) > 2 {
				pidName = append(pidName, PidName{slice[0], slice[2][1:]})
			}
		}
		InsertModule(pidName)
		timePrint("[INFO] Initialize Watcher...")
		// Set beforeHash to the current hash at initialization
		beforeHash = GetHash(string(data))
	} else {
		fmt.Println("There are no running containers in the current namespace.\nExit...")
		_, err := exec.Command("sh", "-c", "rmmod metric.ko").Output()
		if err != nil {
			fmt.Printf("Failed to remove module: %v\n", err)
		}
		os.Exit(0)
	}

	SetupSignalHandling()
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

	InitWatcher()
	var wg sync.WaitGroup
	wg.Add(1)

	go ContainerChangeDetector(&wg)
	wg.Wait()
}
