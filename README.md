# eBPF mon

리눅스 호스트에서 작동 중인 컨테이들의 자원 사용량을 매우 낮은 오버헤드로 측정합니다.

## 실행 방법

리눅스 커널 모듈을 기반으로 작동합니다. WSL2 환경에서도 사용할 수 있지만, 별도의 커널 빌드 절차가 필요합니다.

> 본 모듈은 커널 버전 5.15.0 이외의 환경에서 테스트되지 않았습니다. 커널 버전이 일치하지 않을 경우, 커널 모듈 삽입 시 커널 패닉이 발생할 수 있습니다.

### 필요 도구 설치

빌드에 필요한 도구들을 설치합니다. 커널 모듈을 제어하는 `manager`도 함께 사용하려면, Go 1.16 이상의 버전이 필요합니다. Go 공식 홈페이지에서 직접 설치하거나, 아래 명령어에서 사용된 golang-tools-install-script로 편리하게 설치할 수 있습니다.

```shell
sudo apt install build-essential
# Golang install script
wget -q -O - https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh | bash
```

### 자원 사용량 측정용 커널 모듈

1. 레포지토리 클론

```shell
git clone https://github.com/yidoyoon/ebpf-mon.git
```

2. 커널 모듈 빌드

자원 사용량 수집을 위한 리눅스 커널 모듈을 생성하고, `manager`를 빌드합니다. 

```shell
cd ebpf-mon
make build
```

3. 커널 모듈 로드

빌드한 커널 모듈을 로드합니다. `manager`는 측정 대상의 변화에 따라, 새로운 컨테이너 정보를 반영한 커널 모듈을 자동으로 삽입합니다.

단, 측정 대상 컨테이너가 없다면 일정 시간 대기 후 스스로 종료합니다.

> 커널 모듈 제어에는 root 권한이 필요합니다.

```shell
sudo ./manager
```

다음과 같은 프롬프트에서 테스트용 더미 컨테이너를 생성하는 것도 가능합니다.

```shell
Do you want to create some dummy containers? (y/N)
y
How many containers(busybox)?:
30
```

`lsmod` 명령어를 통해서 커널 모듈을 정상적으로 로드했는지 확인할 수 있습니다.

```shell
lsmod

Module                  Size  Used by
metric                 20480  0
```

아래 명령어로 커널 모듈을 직접 제어하는 것도 가능합니다. 다만, `manager`를 사용하는 것이 훨씬 편리합니다.

**모듈 삽입**

```shell
sudo insmod metric.ko pid=<PID> container_name=<CONTAINER_NAME> pid_count=<PID_COUNT>
```

**모듈 제거**

측정 대상을 변경하려면, 커널 모듈을 우선 제거 후 다시 로드해야합니다. 커널 모듈은 아래 명령어로 제거할 수 있습니다.

```shell
sudo rmmod metric
```

4. 자원 사용량 출력

아래 명령어로 메트릭을 출력합니다.

```shell
cat /proc/metric
```

출력 데이터는 아래와 같은 형태로 출력됩니다. 현재 버전은 메모리 관련 정보만 수집하도록 구현됐습니다.

```shell
<컨테이너 이름>_<자원 종류> <값>
```

예를 들어 `busybox1`이라는 이름을 가진 컨테이너를 대상으로 측정하는 경우,

```shell
busybox1_cache 2969600
busybox1_rss 151552
busybox1_rss_huge 0
busybox1_shmem 0
busybox1_mapped_file 2220032
busybox1_dirty 0
busybox1_writeback 0
busybox1_swap 0
busybox1_pgpgin 1509
busybox1_pgpgout 747
```

### 벤치마크

다른 컨테이너 자원 사용량 수집 도구인 [cAdvisor](https://github.com/google/cadvisor)와 성능 비교를 수행할 수 있습니다. 다만, cAdvisor는 기본적으로 수집하는 항목이 훨씬 많기 때문에, 비슷한 조건에서 성능 비교를 수행할 수 있도록 수집 대상을 제한하는 것이 필요합니다. [cadvisor-lite](https://github.com/yidoyoon/cadvisor-lite)는 벤치마크를 수행하기 위해 기존의 cAdvisor 레포지토리를 수정하여 수집 항목을 최적화한 라이브러리입니다.

(물론, 원본 cAdvisor와도 벤치마크를 수행할 수 있습니다. 이 경우엔 직접 [https://github.com/google/cadvisor](https://github.com/google/cadvisor)를 클론해서 빌드 후 벤치마크를 실행합니다.)

cAdvisor-lite는 서브 모듈로 등록되어 있습니다. 아래 명령어를 통해 서브 모듈을 클론하고 빌드를 진행합니다.

```shell
make setup_benchmark
```

이후 `utils/cadvisor-lite`경로에서 추가 플래그와 수집 대상을 최소화하여 cAdvisor-lite를 실행합니다. root 권한과 함께 실행하지 않는다면, 메트릭 데이터가 수집되지 않을 수 있습니다.

```shell
# Run cAdvisor-lite with minimum stats
cd ./utils/cadvisor-lite/_output
sudo ./cadvisor-lite --disable_metrics=advtcp,app,cpu,cpuLoad,cpu_topology,cpuset,disk,diskIO,hugetlb,memory,memory_numa,network,oom_event,percpu,perf_event,process,referenced_memory,resctrl,sched,tcp,udp,memory_numa,referenced_memory --enable_metrics=memory --docker_only=true
```

cAdvisor-lite가 정상적으로 실행 중인지 확인하려면, 아래 명령어를 통해 메트릭을 리턴하고 있는지 확인합니다. 모니터링 대상이 없다면 `{}`를 출력합니다.

```shell
curl "localhost:8080/api/v2.0/stats?type=docker&recursive=true&count=1"
```

이후 프로젝트 루트의 `benchmark` 디렉토리에서 아래 명령어로 벤치마크를 실행합니다. 리눅스 커널 모듈과 cAdvisor가 호스트에서 구동중이어야 합니다. `COUNT`는 모니터링 대상으로 생성할 더미 컨테이너(busybox) 갯수를 설정하는 환경 변수입니다.

```shell
# Build cAdvisor-lite
cd ../../../benchmark
COUNT=200 go test -bench=. -benchmem
```

벤치마크 실행 후 아래와 같은 결과가 출력됩니다. 벤치마크가 정상적으로 마치면, 벤치마크를 위해 생성했던 더미 컨테이너는 자동으로 제거됩니다.

```shell
goos: linux
goarch: amd64
pkg: github.com/yidoyoon/ebpf-mon/benchmark
cpu: 11th Gen Intel(R) Core(TM) i7-11800H @ 2.30GHz
Benchmark/kernelModule-8                    1020           1151839 ns/op          469438 B/op         42 allocs/op
Benchmark/cadvisorClient-8                    60          17224984 ns/op         1308481 B/op       5636 allocs/op

Test is done. Clean up dummy containers...
PASS
ok      github.com/yidoyoon/ebpf-mon/benchmark  93.865s
```

## License

Distributed under the `GPL-3.0`
