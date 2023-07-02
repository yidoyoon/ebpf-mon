#include <linux/uaccess.h>
#include <linux/kernel.h>
#include <linux/module.h>
#include <linux/errno.h>
#include <linux/kthread.h>
#include <linux/freezer.h>
#include <linux/slab.h>
#include <linux/delay.h>
#include <linux/seq_file.h>
#include <linux/init.h>
#include <linux/pid.h>
#include <linux/sched.h>
#include <linux/proc_fs.h>
#include <linux/build_bug.h>
#include <linux/mm.h>
#include <linux/memcontrol.h>
#include <linux/cgroup.h>
#include <linux/mmzone.h>

#define FILE_NAME "metric"
#define BUF_SIZE 512

char kernel_buf[BUF_SIZE];
pid_t pid[BUF_SIZE];
int pid_count = 0;
static char* container_name[BUF_SIZE];
static int number_of_elements = 0;

module_param(pid_count, int, 0);
module_param_array(pid, int, NULL, 0);
module_param_array(container_name, charp, &number_of_elements, 0644);

MODULE_PARM_DESC(container_name, "The processID given as input from user");
MODULE_PARM_DESC(pid, "The processID given as input from user");

struct mm_struct* mm;
struct task_struct* task;

static const unsigned int memcg1_stats[] = {
	NR_FILE_PAGES,
	NR_ANON_MAPPED,
#ifdef CONFIG_TRANSPARENT_HUGEPAGE
	NR_ANON_THPS,
#endif
	NR_SHMEM,
	NR_FILE_MAPPED,
	NR_FILE_DIRTY,
	NR_WRITEBACK,
	MEMCG_SWAP,
};

static const unsigned int memcg1_events[] = {
   PGPGIN,
   PGPGOUT,
   PGFAULT,
   PGMAJFAULT,
};

static const char *const memcg1_stat_names[] = {
	"cache",
	"rss",
#ifdef CONFIG_TRANSPARENT_HUGEPAGE
	"rss_huge",
#endif
	"shmem",
	"mapped_file",
	"dirty",
	"writeback",
	"swap",
};

static const char* const memcg1_event_names[] = {
   "pgpgin",
   "pgpgout",
   "pgfault",
   "pgmajfault",
};

static unsigned long memcg_events_local(struct mem_cgroup* memcg, int event) {
	long x = 0;
	int cpu;

	for_each_possible_cpu(cpu)
		x += per_cpu(memcg->vmstats_percpu->events[event], cpu);

	return x;
}

static int procmon_proc_show(struct seq_file* m, void* v) {
	unsigned int q;
	for (q = 0; q < pid_count; q++) {
		task = pid_task(find_vpid(pid[q]), PIDTYPE_PID);

        if (!task) {
            pr_err("Could not find task for pid %u\n", pid[q]);
            continue;
        }
        mm = task->mm;
        if (!mm) {
            pr_err("Could not find memory map for task %p\n", task);
            continue;
        }
        if (q >= ARRAY_SIZE(container_name)) {
            pr_err("Index %u is out of bounds for container_name\n", q);
            continue;
        }

		struct mem_cgroup* memcg = get_mem_cgroup_from_mm(mm);
		unsigned int i;
		unsigned long long total_vm;

		total_vm = mm->total_vm;

		// cgroup memory
		for (i = 0; i < ARRAY_SIZE(memcg1_stats); i++) {
			seq_printf(m, "%s_%s %lu\n", container_name[q], memcg1_stat_names[i], memcg_page_state(memcg, memcg1_stats[i]) * PAGE_SIZE);
		}

		// cgroup pid
		for (i = 0; i < ARRAY_SIZE(memcg1_events); i++) {
			seq_printf(m, "%s_%s %lu\n", container_name[q], memcg1_event_names[i], memcg_events_local(memcg, memcg1_events[i]));
		}

		// TODO: 더 많은 지표 확장 필요
		// 참고: https://github.com/torvalds/linux/blob/v5.15/fs/proc/task_mmu.c
		// /proc/[pid]/status
		seq_printf(m, "%s_VmSize %llu\n", container_name[q], (total_vm << (PAGE_SHIFT-10)));
	}

	return 0;
}

static int procmon_proc_open(struct inode* inode, struct file* file) {
	return single_open(file, procmon_proc_show, NULL);
}

static ssize_t procmon_proc_write(struct file* file, const char __user* buf, size_t count, loff_t* ppos) {
    memset(kernel_buf, 0, BUF_SIZE);

	if (count > BUF_SIZE) {
		count = BUF_SIZE;
	}

	if (copy_from_user(buf, buf, count)) {
		return -EFAULT;
	}

    printk(KERN_INFO "proc write : %s\n", kernel_buf);
	return (ssize_t)count;
}

struct proc_ops fops = {
   .proc_open = procmon_proc_open,
   .proc_read = seq_read,
   .proc_lseek = seq_lseek,
   .proc_write = procmon_proc_write,
   .proc_release = single_release,
};

static int __init init_procmon(void) {
	struct proc_dir_entry* procmon_proc;
	procmon_proc = proc_create(FILE_NAME, 644, NULL, &fops);

	if (!procmon_proc) {
		printk(KERN_ERR "Cannot create procmon proc entry \n");
		return -1;
	}
	printk(KERN_INFO "[INFO] Metric Module ON\n");

	return 0;
}

static void __exit exit_procmon(void) {
	remove_proc_entry(FILE_NAME, NULL);
	printk(KERN_INFO "[WARN] Metric Module OFF\n");
}

module_init(init_procmon);
module_exit(exit_procmon);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Sang-Hoon Choi <csh0052@gmail.com>");
MODULE_AUTHOR("Doyoon Yi <yidoyoon@yidoyoon.com>");
MODULE_DESCRIPTION("Memory Metric Monitoring Module");
MODULE_DESCRIPTION("The kernel version must be 5.15.0.");
