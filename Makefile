obj-m += metric.o

GO_FILES=$(shell pwd)/util/main.go
EXECUTABLE_NAME=watcher
TMP=container_info

all: build_module build_go

build_module:
	make -C /lib/modules/$(shell uname -r) M=$(shell pwd) modules

build_go:
	@if [ ! -f go.mod ]; then \
		go mod init github.com/yidoyoon/ebpf-mon; \
	fi
	go mod tidy
	go build -o $(shell pwd)/$(EXECUTABLE_NAME) $(GO_FILES)

clean:
	make -C /lib/modules/$(shell uname -r) M=$(shell pwd) clean
	rm -f $(shell pwd)/$(EXECUTABLE_NAME)
	rm -f $(shell pwd)/$(TMP)
