# eBPF mon

리눅스 호스트 내부에서 작동중인 컨테이들의 자원 사용량을 매우 낮은 오버헤드로 측정합니다. 개선 단계에 있으며, 기존 레포지토리는 (https://github.com/keti-openfx/OpenFx-metering)입니다.

> 리눅스 커널 5.15.0 버전이 필요합니다.

## 실행 방법

### 사전 도구 설치

모듈 빌드에 필요한 도구들을 설치합니다.

```shell
sudo apt install build-essential
```

### 자원 사용량 측정용 커널 모듈

1. 레포지토리 클론

```shell
git clone https://github.com/yidoyoon/ebpf_mon.git
```

2. 커널 모듈 빌드

```shell
cd ebpf_mon
sudo make
```

3. 커널 모듈 로드

빌드한 커널 모듈을 불러옵니다.

- `pid`: 측정 대상 단일 pid 혹은 pid 배열
- `container_name`: 측정 대상 컨테이너 이름 혹은 이름 배열
- `pid_count`: 측정 대상 컨테이너의 갯수

```shell
sudo insmod metric.ko pid=<PID> container_name=<CONTAINER_NAME> pid_count=<PID_COUNT>
```

단일 대상 모니터링

```shell
sudo insmod metric.ko pid=1234 container_name=mycontainer pid_count=1
```

다수 대상 모니터링

```shell
sudo insmod metric.ko pid=1234,5678 container_name=mycontainer,anothercontainer pid_count=2
```

정상적으로 적재되면 `/proc`에서 `metric` 파일을 확인할 수 있습니다.

```shell
find /proc -name "metric"
```

4. 자원 사용량 출력

아래 명령어로 메트릭을 출력합니다.

```shell
cat /proc/metric
```

출력 데이터는 아래와 같은 형태로 출력됩니다.

```shell
<컨테이너 이름>_<자원 종류> <값>
```

예를 들어 busybox 컨테이너를 대상으로 측정하는 경우,

```shell
busybox_cache 2969600
busybox_rss 151552
busybox_rss_huge 0
busybox_shmem 0
busybox_mapped_file 2220032
busybox_dirty 0
busybox_writeback 0
busybox_swap 0
busybox_pgpgin 1509
busybox_pgpgout 747
```

5. 커널 모듈 제거

커널 모듈에 변경 사항이 필요하면 모듈을 먼저 제거한 뒤 다시 불러와야 합니다.

```
sudo rmmod metric.ko
```

### 커널 모듈 제어기

WIP

참고(https://github.com/keti-openfx/OpenFx-metering/blob/master/core/kenel_manager.py)

## License

Distributed under the `GPL-3.0`
