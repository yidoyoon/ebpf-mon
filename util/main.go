package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
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
			exec.Command("sh", "-c", "rmmod metric.ko").Output()
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

	_, err := exec.Command("sh", "-c", "rmmod metric.ko").Output()
	if err != nil {
		log.Println(err)
	}

	cmd := "insmod metric.ko pid=" + pids[:len(pids)-1] + " pid_count=" + fmt.Sprint(pidLen) + " container_name=" + names[:len(names)-1]

	_, err = exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Println(err)
	}

	timePrint("[INFO] Container count(s) : " + fmt.Sprint(pidLen))
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
				pidName = append(pidName, PidName{slice[0], slice[2]})
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
	timePrint("[INFO] Initialize Watcher...")
	var pidName []PidName

	containerData := "./.container_info"
	exec.Command("sh", "-c", "docker ps -q|xargs docker inspect --format '{{.State.Pid}},{{.ID}},{{.Name}}' > "+containerData).Output()
	data, _ := os.ReadFile(containerData)
	trimmedData := strings.TrimSpace(string(data))
	containerList := strings.Split(trimmedData, "\n")
	for _, container := range containerList {
		slice := strings.Split(container, ",")
		pidName = append(pidName, PidName{slice[0], slice[2]})
	}
	InsertModule(pidName)
}

func CreateTestContainer() {
	fmt.Println("How many containers(busybox)?: ")
	var count int
	var discard string
	fmt.Scanf("%d\n", &count)
	fmt.Scanln(&discard)
	for i := 0; i < count; i++ {
		exec.Command("sh", "-c", "docker run -dt ubuntu").Output()
	}
}

func main() {
	SetupSignalHandling()
	fmt.Println("Do you want to create some dummy containers? (y/N)")
	var command string
	_, err := fmt.Scanln(&command)

	for err != nil {
		fmt.Println("The input is not valid. Please type 'y' for yes, 'N' for no.")
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
