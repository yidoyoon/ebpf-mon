obj-m += metric.o

WATCHER_SRC=$(shell pwd)/monitoring_manager/manager.go
WATCHER_BIN=$(shell pwd)/manager
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
	git submodule update --init
	$(MAKE) -C ./utils/cadvisor-lite build

clean:
	make -C /lib/modules/$(shell uname -r) M=$(shell pwd) clean
	rm -f $(WATCHER_BIN)
	rm -f $(CONTAINER_INFO)

.PHONY: clean
