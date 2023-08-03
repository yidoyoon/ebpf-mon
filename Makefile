obj-m += metric.o

WATCHER_SRC=$(shell pwd)/watcher/watcher.go
WATCHER_BIN=$(shell pwd)/watcher/watcher
PERF_STAT_SRC=$(shell pwd)/utils/read_proc/read_proc.go
PERF_STAT_BIN=$(shell pwd)/utils/read_proc/read_proc
CONTAINER_INFO=$(shell pwd)/container_info

build: build_module build_watcher
setup_benchmark: build_cadvisor_lite build_module build_watcher

build_module:
	make -C /lib/modules/$(shell uname -r) M=$(shell pwd) modules

build_watcher:
	@if [ ! -f go.mod ]; then \
		go mod init github.com/yidoyoon/ebpf-mon; \
	fi
	go mod tidy
	go build -o $(WATCHER_BIN) $(WATCHER_SRC)

build_cadvisor_lite:
	@ifeq [ $(wildcard ./cadvisor-lite/.*), ]
		$(MAKE) -C ./cadvisor-lite build
	@else
		@echo "Please clone submodule with below command first."
		@echo "git submodule update --init"

#build_perf_stat:
#	go build -o $(PERF_STAT_BIN) $(PERF_STAT_SRC)

clean:
	make -C /lib/modules/$(shell uname -r) M=$(shell pwd) clean
	rm -f $(WATCHER_BIN)
	rm -f $(CONTAINER_INFO)

.PHONY: clean
