## 用到的C库函数,大多数函数可以参考http://www.man7.org/linux/man-pages/dir_all_alphabetic.html。

### void _exit(int status);
#include <unistd.h>  
立即中止当前进程，关闭当前进程的所有文件描述符，进程的子进程由进程1，init继承，并给该进程的父进程发送一个SIGCHLD信号，系统默认将忽略此信号，如果父进程希望被告知其子系统的这种状态，则应捕捉此信号。
status会作为退出状态返回给父进程，可以被wait(2)族的调用收集。

### void exit(int status);
#include <stdlib.h>  
该函数会中止当前进程，并将status & 0377返回给父进程。  
所有使用atexit(3)和on_exit(3)注册的函数会按照与注册相反的顺序调用。所有的stdio(3)会被flush和close。使用tmpfile(3)创建的文件会被删除。
C语言标准指定了两个常量EXIT_SUCCESS和EXIT_FAILURE，可以传给exit()，用来指示是成功还是失败地中止。

### int atexit(void (*function)(void));
#include <stdlib.h>  
atexit()注册在进程终止时候调用的函数。注册的函数没有参数。终止要么通过exit(3)要么从main函数返回。调用顺序和注册顺序相反。同一个函数可以注册多次。注册的数量限制可以通过sysconf(3)T获取。当通过fork(2)创建一个子进程时，子进程会继承父进程的注册信息。在成功调用一个exec(3)函数后，所有的注册都被移除。

### int on_exit(void (*function)(int , void *), void *arg);
功能和atexit一样，除了注册的函数会被传入最后调用exit(3)时的status和一个on_exit()的arg参数。

### getpid() 
#include <unistd.h>  
返回进程的ID(PID)。经常会被用于产生临时文件，以减少临时文件名冲突。

### int gettimeofday(struct timeval *tv, struct timezone *tz);  
#include <sys/time.h>  
该函数会获取系统时间并设置到传入参数上。timeval的结构如下:  

    struct timeval {
        time_t      tv_sec;     /* seconds */
        suseconds_t tv_usec;    /* microseconds */
    };
    分别为从Epoch以来的秒数和微妙数。
timezone结构的使用已经obsolete,tz参数通常应该设置为NULL。

### char *setlocale(int category, const char *locale);
该函数用来设置或者查询程序当前的locale。  
#include <locale.h>  
- category代表类别，比如LC_COLLATE代表字符串排序。  
- locale是对应类别的具体字符串。  
  如果locale是空字符串。会根据环境变量来设置locale。各个实现可能会不一样，对glibc首先会检查LC_ALL，其次会检查和category一样名字的环境变量，最后会检查环境变量LANG。会使用最先找到的环境变量。  
  如果locale是NULL，那么就不会修改locale，而只是查询。

locale "C"或者"POSIX"是可移植的。在程序启动的时候，可移植的locale "C"被用来作为默认值。
所以如果直接char *collate = setlocale(LC_COLLATE,NULL); 会得到"C"。  
在setlocale(LC_COLLATE,"")后再collate = setlocale(LC_COLLATE,NULL); 会得到系统的collate设置，比如"en_US.UTF-8"。  
一个程序可以通过setlocale(LC_ALL, "");来做到可移植。

### void tzset (void);
#include <time.h>
extern char *tzname[2];  
extern long timezone;  
extern int daylight;  
tzset()使用TZ环境变量来初始化tzname变量。该函数会被其他依赖时区的时间转换函数自动调用。在类System-V环境，同时会设置timezone和daylight变量。如果没有TZ环境变量，则会使用系统时区。系统时区是一个使用tzfile(5)格式的文件/etc/localtime。
           
### size_t strftime(char *s, size_t max, const char *format,const struct tm *tm);
#include <time.h>  
该函数根据格式字符串来格式化时间信息tm，并将结果放到长度最大为max的字符数组s中。

### int fflush(FILE *stream);
#include <stdio.h>  
冲刷一个流。
- 对输出流，fflush()对给定的输出的所有的用户空间的缓冲数据强制进行一次write，或者通过流的底层write函数更新流。如果参数为NULL，那么所有打开的输出流都会被冲刷。
- 对和可寻址文件(例如磁盘文件，但是不包括管道和终端)，fflush()丢弃所有已经从底层文件获取到，但是还未被应用消费的缓冲数据。

### void syslog(int priority, const char *format, ...);
#include <syslog.h>  
syslog()产生一条日志消息，该消息会由syslogd(8)分发。priority由facility值或level值而来。如果priority不包含或的facility值，则会使用由openlog()设置的默认值，如果前面没有调用openlog(),则会使用默认的LOG_USER。剩下的参数为一个和printf(3)一样的(除了%m会被strerror(errno)产生的错误消息替换)格式化字符串format和对应参数。format不需要包含结束的换行符。  
注意：永远不要在format中使用用户提供的字符串，应该使用如下形式：syslog(priority, "%s", string); 这里应该不是说只应该用"%s",而是说不应该直接使用用户提供的字符串作为format，类似不要在SQL中直接拼接用户提供的数据集的意思。
redis中使用如下：
        //LOG_PID 每条日志信息中都包括进程号
        // LOG_NDELAY 立即打开与系统日志的连接（通常情况下，只有在产生第一条日志信息的情况下才会打开与日志系统的连接）
        // LOG_NOWAIT 在记录日志信息时，不等待可能的子进程的创建
        openlog(server.syslog_ident, LOG_PID | LOG_NDELAY | LOG_NOWAIT,server.syslog_facility);

### time_t time(time_t *tloc);
#include <time.h>  
获取从Epoch以来的秒数，如果tloc不是NULL，那么同样会将该秒数设置到tloc中。  

### void srand(unsigned int seed);
设置随机数生成器的种子，对于同一个种子，每次生成的整数序列都是一样的。

### char *strstr(const char *haystack, const char *needle);
#include <string.h>  
从haystack中找到子串needle第一次出现的位置，不比较结尾的null。如果没有找到返回NULL。

### void *realloc(void *ptr, size_t size);
#include <stdlib.h>  
realloc()修改由ptr指向的内存块的大小。从开始位置到老的和新的大小的最小值之间的内容保持不变，新增的内存不会被初始化。  
如果ptr是NULL，那么等价于malloc(size)。如果size为0，并且ptr不为NULL，那么等价于free(ptr)。除非ptr是NULL，否则其必须是之前调用malloc(), calloc(), 或者 realloc()返回的。

###  int fcntl(int fd, int cmd, ... /* arg */ );
#include <unistd.h>  
#include <fcntl.h>  
操作文件描述符。fcntl()对文件描述符fd进行cmd操作。根据cmd不同，可能会有arg参数。
有一些操作只在特定linux内核版本之后才能用，调用 fcntl()的时候，可以检查调用是否因为EINVAL失败，如果是的话，则表明不支持该操作。
下面是遇到的一些操作：

        F_GETFL (void)
              返回文件访问模式和文件状态标志。

        F_SETFL (int)
              设置文件状态标志。文件访问模式(O_RDONLY, O_WRONLY, O_RDWR)和一些特定文件标志(例如, O_CREAT, O_EXCL, O_NOCTTY, O_TRUNC)会被忽略。
              在Linux下，该命令只能修改O_APPEND,O_ASYNC, O_DIRECT, O_NOATIME 和 O_NONBLOCK flags.  

O_NONBLOCK标记  
只要可能，文件在非阻塞模式下打开，无论是open()还是后续的I/O操作都不会造成调用进程阻塞。
注意，该标记对poll(2), select(2), epoll(7)及类似操作没有影响，因为这些接口只是通知调用方是否有一个文件描述符可用，可用意味着I/O操作不设置O_NONBLOCK也不会阻塞。
注意，该标记对普通文件和块设备不起作用，也就是说无论是否设置O_NONBLOCK，只要需要设备活动，I/O操作就会阻塞。
O_NONBLOCK最终也可能会被实现，当对普通文件和块设备设置该标记，应用不应该依赖阻塞行为。
设置该标记的代码大概如下（忽略错误处理）：
    int flags;
    flags = fcntl(fd, F_GETFL);
    if (non_block)
        flags |= O_NONBLOCK;
    else
        flags &= ~O_NONBLOCK;
    fcntl(fd, F_SETFL, flags);

### int setsockopt(int sockfd, int level, int optname, const void *optval, socklen_t optlen);
#include <sys/types.h>
#include <sys/socket.h>  
setsockopt()操作代表socket的文件描述符sockfd的选项。选项可以存在于多个协议层次，它们永远存在于最高的socket层。
当操作socket选项时，必须指定选项的名称和选项所在的层级。为了在socket接口层面操作选项，层级需要指定为SOL_SOCKET。
如果要指定一个选项在TCP协议中被解析，层级应该设置为TCP的协议号码，协议号码请参考getprotoent()。
optval和optlen用于传递选项值。
Optname和指定的选项被未经解析的传递到相应的协议模块再做解析。
头文件<sys/socket.h>包含socket层级的选项定义。其它协议层级的选项会有不同的格式和名称。
大多数socket层级的选项使用int类型的optval，对于布尔型选项，使用0关闭选项，使用非零启用选项。
socket选项可以参考socket(7)。  

TCP_NODELAY选项  TCP_NODELAY是用来禁用Nagle算法的  

Nagle算法是为了减少广域网的小分组数目，从而减小网络拥塞的出现。默认是开启的。
该算法要求一个tcp连接上最多只能有一个未被确认的未完成的小分组，在该分组ack到达之前不能发送其他的小分组，tcp需要收集这些少量的分组，并在ack到来时以一个分组的方式发送出去。
其中小分组的定义是小于MSS的任何分组。  
对于实时性要求比较高的应用，应该开启TCP_NODELAY选项以关闭Nagle算法。  
使用方式如：  
        int val = 1;
        setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &val, sizeof(val))  

SO_KEEPALIVE  在面向连接的socket中激活keep-alive。

keep-alive是TCP的连接检测机制，涉及到以下参数：

- tcp_keepalive_intvl (integer; default: 75; since Linux 2.4)
              两次TCP keep-alive探测的时间间隔，默认75s.
- tcp_keepalive_probes (integer; default: 9; since Linux 2.2)
              TCP keep-alive探测的最大次数，默认9。超过这个次数没有得到响应，就放弃并杀死连接。

- tcp_keepalive_time (integer; default: 7200; since Linux 2.2)
              TCP开始发送keep-alive探测前，连接需要空闲的秒数。
              Keep-alives只在SO_KEEPALIVE选项开启的时候才会发送。默认值是7200s(2小时)。
              一个空闲连接大概在11分钟后会被终止(每75s进行一次探测，共进行9次).
              注意底层的连接跟踪机制和应用超时可能会比这些设置短很多。

SO_REUSEADDR 表明在bind(2)中用来确认地址的规则应该允许重用本地地址。
             对AF_INET来说，这意味着除非已经有一个活跃监听socket绑定到了地址上，那么socket就可以绑定。
             当监听socket绑定到INADDR_ANY和一个指定端口，那么就不能再绑定该端口到任一本地地址。

             Linux只在执行bind(2)绑定到某个端口的之前的程序，和想要重用端口的程序都设置了SO_REUSEADDR才允许端口重用。
             也就是说，如果不是INADDR_ANY，那么该选项允许同一个端口绑定到不同的本地地址上。

             另外，因为TIME_WAIT会导致立即重启服务时报Address already in use错误。设置该选项可以保证在重启时，将原先端口抢过来，不会再报地址已在使用错误。
             所有TCP服务器都应该指定本套接字选项，以允许服务器在这种情形下被重新启动。

redis中使用如下:  

        int val = 1;
        setsockopt(fd, SOL_SOCKET, SO_KEEPALIVE, &val, sizeof(val) // 开启SO_KEEPALIVE
        val = interval; // inverval为配置参数
        setsockopt(fd, IPPROTO_TCP, TCP_KEEPIDLE, &val, sizeof(val);

        val = interval/3;
        if (val == 0) val = 1;
        setsockopt(fd, IPPROTO_TCP, TCP_KEEPINTVL, &val, sizeof(val);

        /* Consider the socket in error state after three we send three ACK
        * probes without getting a reply. */
        val = 3;
        setsockopt(fd, IPPROTO_TCP, TCP_KEEPCNT, &val, sizeof(val));

### int socket(int domain, int type, int protocol);
#include <sys/types.h>  
#include <sys/socket.h>  

socket()为通信创建一个端点，并且返回一个指向该端点的文件描述符。
返回的文件描述符会是当前进程中未打开的数值最小的文件描述符。
domain参数指定一个通信域；这选择通信用的协议族。
这些协议族定义在<sys/socket.h>中。
- AF_UNIX和AF_LOCAL 是一样的，指本地通信。
- AF_INET  IPV4网络协议
- AF_INET6 IPV6网络协议

type指定了通信的语义：
- SOCK_STREAM 提供顺序的，可靠的，双向的，面向连接的字节流。也可能支持带外数据传输。
- SOCK_DGRAM  支持数据报(无连接，最大大小固定的不可靠的消息)

protocol指定一个特定的协议。通常对一个给定的协议族一个特定的socket type只对应一个特定的协议，这种情况下，protocol可以指定为0.
然后，也可能存在多种协议，这种情况下就必须指定协议，见protocols(5)和getprotoent(3) 。

### int getrlimit(int resource, struct rlimit *rlim);   int setrlimit(int resource, const struct rlimit *rlim);
#include <sys/time.h>  
#include <sys/resource.h>  
设置资源限制。
每种资源都一个软限制和一个硬限制。其中rlim_cur是操作系统设置的当前限制，rlim_max是rlim_cur能设置的最大值。
           struct rlimit {
               rlim_t rlim_cur;  /* Soft limit */
               rlim_t rlim_max;  /* Hard limit (ceiling for rlim_cur) */
           }
        一个非特权进程可以设置软限制为0到硬限制之间的值，或者不可逆的减少硬限制的值。
        一个特权进程（在Linux下有CAP_SYS_RESOURCE能力的用户）可以随意修改软限制和硬限制。
使用方式:
    struct rlimit limit;
    getrlimit(RLIMIT_NOFILE,&limit);
    limit.rlim_cur = bestlimit;
    limit.rlim_max = bestlimit;
    setrlimit(RLIMIT_NOFILE,&limit);

### epoll I/O事件通知机制

Epoll API监控多个文件描述符，看其中是否有可以进行I/O操作的。
Epoll可以用于边缘触发或者水平触发。
Epoll API的核心概念是一个内核数据结构epoll实例，从用户空间来看，相当于是一个包含两个列表的容器：
    - 兴趣列表（有时也叫做epoll集合）：进程注册的需要监控的文件描述符集合。
    - 就绪列表：已经可以进行I/O操作的文件描述符。就绪列表是兴趣列表的一个子集，由内核根据文件描述符的I/O活动动态设置。  

Linux提供了下列系统调用来创建和管理epoll实例：

       - epoll_create(2) 创建一个epoll实例并且返回一个对引用该实例的文件描述符。更新的epoll_create1(2)对epoll_create(2)进行了功能扩展。

       - 对特定文件描述符的兴趣由epoll_ctl(2)来进行注册。

       -  epoll_wait(2)等待I/O事件，如果没有任何事件则阻塞调用线程。该系统调用可以看做是从epoll实例的就绪列表中获取元素。

### int epoll_ctl(int epfd, int op, int fd, struct epoll_event *event);
#include <sys/epoll.h>
该系统调用用来增加，修改或者删除文件描述符epfd引用的epoll实例的兴趣列表。
它要求对目标文件描述符fd执行op指定的操作，合法的op参数为：It requests that the operation op be performed for
    - EPOLL_CTL_ADD
        将fd加入到兴趣列表中，并且将event指定的设置和指向fd的内部文件关联起来。

    - EPOLL_CTL_MOD
        使用event中指定的新配置修改兴趣列表中关联的fd的设置。

    - EPOLL_CTL_DEL
        从兴趣列表中移除目标文件描述符fd，event参数被忽略并且可以为NULL。

event参数描述了连接到fd的对象，epoll_event结构体的定义为:

           typedef union epoll_data {
               void        *ptr;
               int          fd;
               uint32_t     u32;
               uint64_t     u64;
           } epoll_data_t;

           struct epoll_event {
               uint32_t     events;      /* Epoll events */
               epoll_data_t data;        /* User data variable */
           };
       events字段是一个由event类型OR得到的位掩码，常见的：
        - EPOLLIN
              关联的文件可读了，可以在其上调用read(2).

       - EPOLLOUT
              关联的文件可写了，可以在其上调用write(2).

### int pipe(int pipefd[2]); 创建管道
#include <unistd.h>  
pipe()创建一个管道。  
管道是一个单向的数据通道，可以用来作为进程间通信。pipefd数组用来返回管道的两个端点。
pipefd[0]引用管道的读端，pipefd[1]引用管道的写端。从写端写入数据，从读端读取数据。

如果一个进程试图从空管道中读取数据，那么read(2)会阻塞直到有数据可用。
如果一个进程试图往一个满的管道中写数据，那么write(2)会阻塞直到足够的数据被从管道中读取了从而允许完成写操作。
通过使用fcntl(2)的F_SETFL操作设置O_NONBLOCK状态标记，可以使用非阻塞I/O。

通过管道提供的通道是字节流：没有任何消息边界概念。

如果引用管道的写端的所有文件描述符都被关闭了，那么使用read(2)读取管道数据会得到EOF（read(2)会返回0）。
如果引用管道的读端的所有文件描述符都被关闭了，那么使用write(2)读取管道数据会为调用进程产生一个SIGPIPE信号。
如果调用进程忽略这个信息，write(2)会以错误EPIPE结束。
使用pipe(2)和fork(2)的应用应该使用合适的close(2)调用关闭不必要的重复文件描述符；这样可以保证EOF和SIGPIPE/EPIPE在合适的时候被传送。
不能对管道使用lseek(2)。

管道容量
    管道有个受限的容量。如果管道满了，那么取决于O_NONBLOCK标记是否设置，write(2)会阻塞或失败。
    不同的实现有不同的限制。应用不应该依赖特定的容量：应用应该设计为读进程尽可能快的消费数据，以防止写进程阻塞。
    从Linux 2.6.35开始,默认的管道容量为16个页(在一个页大小为4096字节的系统共65,536字节)，但是可以通过fcntl(2)的F_GETPIPE_SZ和F_SETPIPE_SZ来获取和设置容量。
    ioctl(fd, FIONREAD, &nbytes);用于管道的任一端，可以获取管道中未读的字节数，然后放置到nbytes中。

PIPE_BUF
       POSIX.1要求write(2)写入少于PIPE_BUF的字节时必须是原子的：写入管道的数据为一个连续的序列。
       写入超过PIPE_BUF数据可以是非原子的:内核可能将写入的数据和别的进程的写入数据交错写入。
       POSIX.1要求PIPE_BUF最少是512字节.  在Linux,PIPE_BUF是4096字节.
       精确的语义取决于是否设置了O_NONBLOCK，是否有多个写入进程还有写入的字节数：

       O_NONBLOCK disabled, n <= PIPE_BUF
              所有n字节原子写入；如果没有足够的空间，write(2)可能会阻塞。

       O_NONBLOCK enabled, n <= PIPE_BUF
              如果有空间可以写入n字节，那么write(2)立即成功写入所有n字节；否则write(2)失败，设置errno为EAGAIN。

       O_NONBLOCK disabled, n > PIPE_BUF
              写入是非原子的：write(2)写入的数据可能和别的进程的write(2)写入的数据交织在一起；write(2)阻塞直到所有n字节都被写入。

       O_NONBLOCK enabled, n > PIPE_BUF
              如果管道满了，那么write(2)会失败,并将errno设置为EAGAIN. 
              否则，1到n字节可能被写入（也就是说，可能发生部分写入；调用者应该检查write(2)的返回值，以查看多少字节实际被写入了），
              另外,写入的字节也可能和别的线程写入的字节交织在一起。

打开文件状态标记

    对于管道，只有O_NONBLOCK和O_ASYNC有意义。
    在管道的读端设置O_ASYNC造成当管道上有新输入时产生一个信号（默认为SIGIO）。
    信号递送的目标必须使用fcntl(2) F_SETOWN设置。

### int pthread_mutex_lock(pthread_mutex_t *mutex); 锁定互斥量
#include <pthread.h>  
如果系统提供的默认属性已经够用，那么可以使用PTHREAD_MUTEX_INITIALIZER宏来初始化一个静态分配的mutex。
如果pthread_mutex_lock()返回0或者[EOWNERDEAD]，那么mutex对象就被锁定了。
如果mutex已经被另一个线程锁定，那么调用线程应该阻塞直到mutex可用。
该操作返回时，应该返回处于锁定状态的互斥量引用的mutex对象，该对象的所有者应该为调用线程.
如果一个线程试图重新锁定它已经锁定了的mutex，或者一个线程试图释放一个未锁定的mutex或者非由其锁定的mutex，那么根据不同mutex类型和健壮性，其行为见下面表格：

        ┌───────────┬────────────┬────────────────┬───────────────────────┐
        │Mutex Type │ Robustness │     Relock     │ Unlock When Not Owner │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │NORMAL     │ non-robust │ deadlock       │ undefined behavior    │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │NORMAL     │ robust     │ deadlock       │ error returned        │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │ERRORCHECK │ either     │ error returned │ error returned        │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │RECURSIVE  │ either     │ recursive      │ error returned        │
        │           │            │ (see below)    │                       │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │DEFAULT    │ non-robust │ undefined      │ undefined behavior†   │
        │           │            │ behavior†      │                       │
        ├───────────┼────────────┼────────────────┼───────────────────────┤
        │DEFAULT    │ robust     │ undefined      │ error returned        │
        │           │            │ behavior†      │                       │
        └───────────┴────────────┴────────────────┴───────────────────────┘
从表格来看，这个锁是非可重入的。简单起见，不要重新获取已锁定的mutex，也不要释放未锁定的或者非本线程锁定的mutex。

### int pthread_mutex_init(pthread_mutex_t *restrict mutex,const pthread_mutexattr_t *restrict attr);  
#include <pthread.h>  
pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;  
pthread_mutex_init()函数应该使用由attr指定的属性来初始化mutex。
如果attr为NULL，那么会使用默认的mutex属性；效果应该和传递一个默认的mutex属性对象一样。
初始化成功后，mutex的状态变为已初始化并且是未加锁的。

### int pthread_cond_init(pthread_cond_t *restrict cond,const pthread_condattr_t *restrict attr);
#include <pthread.h>  
pthread_cond_t cond = PTHREAD_COND_INITIALIZER;
pthread_cond_init()函数应该使用attr属性来初始化条件cond。
如果attr为NULL，应该使用默认的条件变量属性；效果应该和传递一个默认条件变量属性对象是一样的。
初始化成功后，条件变量应该变成已初始化。  
条件变量和Java的Condition功能类似，使用方式也类似：

    pthread_mutex_lock();
    while (condition_is_false)
        pthread_cond_wait();
    pthread_mutex_unlock();

可以由pthread_cond_signal函数或pthread_cond_broadcast函数唤醒，也可能在被信号中断后被唤醒。

###  int pthread_cond_wait(pthread_cond_t *restrict cond,pthread_mutex_t *restrict mutex);
#include <pthread.h>
pthread_cond_wait()函数在条件变量上阻塞。应用应该保证调用线程在获取mutex锁之后再调用该函数。
该函数原子的释放mutex并且造成调用线程阻塞在条件变量上。

在成功返回后，mutex应该被锁定了，并且由当前调用线程拥有。

当使用条件变量时，永远有一个涉及到关联到每个条件等待的共享变量的Boolean谓词。
可能从pthread_cond_wait()中虚假唤醒。因为pthread_cond_wait()不暗示该谓词的任意事情，该谓词应该在该种返回情况下再次确认。

当一个线程等待一个条件变量，会给pthread_cond_wait()指定一个特定的mutex。
在mutex和条件变量直接会有一个动态绑定，只要最少有一个线程阻塞在条件变量上，该绑定就是有效的。
在此期间，等待该条件变量的线程如果使用了一个不同的mutex，则其效果是未定义的。

条件等待是一个取消点。

当可取消性类型为PTHREAD_CANCEL_DEFERRED时，应对一个取消请求的副作用是，在条件等待时，在调用第一个取消清理处理函数前，重新获取了mutex。
效果就像是线程解除了阻塞，允许执行到从pthread_cond_timedwait()返回，但是就在这时注意到了取消请求，那么不返回到pthread_cond_wait()的调用者，而是开始线程取消行动，包括了调用取消清理函数。

如果一个信号被发送给一个正在等待条件变量的线程，从信号处理函数返回后，线程继续等待条件变量就好像它未被中断一样，或者它应该因为虚假唤醒而返回0.

### int pthread_attr_init(pthread_attr_t *attr);
#include <pthread.h>
pthread_attr_init()函数使用默认属性来初始化attr。
调用完该函数后，可以使用各种相关函数来设置单独的属性，然后该对象可以被用于一个或多个pthread_create(3)调用来创建线程。
当一个线程属性对象不再使用时，应该使用pthread_attr_destroy()函数来将其销毁。
销毁一个进程属性对象对使用该对象创建的线程没有影响。      

### int pthread_attr_setstacksize(pthread_attr_t *attr, size_t stacksize);int pthread_attr_getstacksize(const pthread_attr_t *attr, size_t *stacksize);
#include <pthread.h>
pthread_attr_setstacksize()设置线程属性对象的栈大小属性。
栈大小属性决定了分配给线程的最小值（以字节为单位）。

pthread_attr_getstacksize()函数返回栈大小属性，将其设置到stacksize参数中。

### int pthread_create(pthread_t *thread, const pthread_attr_t *attr,void *(*start_routine) (void *), void *arg);
#include <pthread.h>
pthread_create()函数在调用进程中启动一个新线程。
通过调用start_routine()来启动新线程的执行；arg是传递给start_routine()的唯一参数。
新线程可以以下列的某一种方式结束：
    - 调用pthread_exit(3)，指定退出状态。该状态可由同进程内其它调用pthread_join(3)的线程获得。

    - 从start_routine()返回. 这和使用返回语句的值来调用pthread_exit(3)是一样的。

    - 调用pthread_cancel(3)取消。

    - 进程内的任一线程调用了exit(3),或者主线程从main()中返回了。这会导致进程内的所有线程终止。

attr属性指向一个pthread_attr_t结构体，该结构体的内容在线程创建的时候决定了新线程的属性；该结构体使用和相关函数初始化。
如果attr为NULL，则线程使用默认属性创建。

在返回前，对pthread_create()的成功调用将新线程的ID存储在由thread参数指向的缓冲中。
该ID在后续的pthreads函数中用来引用创建的线程。

新线程继承了调用线程的信号掩码的一份副本(pthread_sigmask(3)).
新线程的挂起信号集合为空（sigpending(2）。
新线程不继承调用线程的信号栈（sigaltstack(2)）。
新线程继承调用线程的浮点数环境(fenv(3))。
新线程的CPU时钟为0（pthread_getcpuclockid(3)）。

特定于Linux的细节：
新线程继承了调用线程的能力集合（capabilities(7)）和CPU亲和性掩码（sched_setaffinity(2)）。

### char *strchr(const char *s, int c);
#include <string.h>  
返回字符c在字符串s中第一次出现处的指针。

### char *getcwd(char *buf, size_t size);
#include <unistd.h>  
返回一个null结尾的字符串，内容为调用进程的当前工作目录的绝对路径。路径名保存到buf中,返回值也就是buf。

###  int sigemptyset(sigset_t *set);
#include <signal.h>  
sigemptyset()将传入的信号集合清空，也就是集合中不再包含任何信号。

### int sigaddset(sigset_t *set, int signum);
#include <signal.h>  
将signum加入到set中。

### int pthread_sigmask(int how, const sigset_t *set, sigset_t *oldset);
#include <signal.h>  
pthread_sigmask()和sigprocmask(2)是一样的,除了POSIX.1标准要求在多线程编程中应该使用pthread_sigmask()。
       
### int sigprocmask(int how, const sigset_t *set, sigset_t *oldset);
#include <signal.h>  
sigprocmask()用来获取或改变调用线程的信号掩码。
信号掩码是其发送被阻塞了的信号集合。
该函数的行为取决于参数how：

    - SIG_BLOCK
        被阻塞的信号集合是当前集合和set参数的合集。

    -  SIG_UNBLOCK
            信号被从当前阻塞的信号集合中移除。可以移除一个未被阻塞的信号。.

    -  SIG_SETMASK
            阻塞信号集合被设置为参数set。
所谓阻塞是指，该信号不会发送，直到后续解除了阻塞。
一个信号可以为作为一个整体的进程产生（例如使用kill(2)产生一个信号）或者为一个指定线程产生（例如， SIGSEGV和SIGFPE或者由pthread_kill(3)产生的信号）。
指向进程的信号可以发送给该进程下没有阻塞该信号的任一线程。如果有多个线程未阻塞该信号，那么内核会选择任意一个线程。

### int sigaction(int signum, const struct sigaction *act, struct sigaction *oldact);
#include <signal.h>  

sigaction()系统调用用来改变收到一个特定信号时进程采用的行动。
Signum可以指定除了SIGKILL和SIGSTOP之外的任意信号。
如果act非NULL，那么处理信号的新的动作由act指定。
如果oldact非NULL，那么之前的动作保存在oldact中。

sigaction会定义为类似如下的结构:

           struct sigaction {
               void     (*sa_handler)(int);
               void     (*sa_sigaction)(int, siginfo_t *, void *);
               sigset_t   sa_mask;
               int        sa_flags;
               void     (*sa_restorer)(void);
           };
在某些架构下，会使用union:所以不要同时给sa_handler和sa_sigaction赋值。
sa_handler指定关联特定信号的行动，可以是SIG_DFL（默认行动），SIG_IGN（忽略该信号），或者一个信号处理函数指针。该函数使用信号作为唯一的参数。

如果在sa_flags指定了SA_SIGINFO ，那么使用sa_sigaction而不是sa_handler来指定信号处理函数。

sa_sigaction的签名为void handler(int sig, siginfo_t *info, void *ucontext)
   - sig为信号
   - info类型为siginfo_t结构，包含了信号的很多信息。
   - ucontext是一个指向ucontext_t结构体的指针, 强制转换为void *.
      该字段包含由内核存储在用户空间堆栈的的信号上下文信息。请参考sigreturn(2)和getcontext(3)。

sa_mask指定了在信号处理函数执行期间会被阻塞的信号。sa_mask必须使用sigemptyset(3)初始化. 

sa_flags指定会改变信号行为的标志集合，其是由一些标志运行或操作得来：

        SA_NODEFER
            不阻止信号在其自身的信号处理器中再次收到同样信号。该标志只在建立信号处理函数时有意义。
        SA_RESETHAND
            在进入信号处理函数后，将信号行动恢复为默认值。该标志只在建立信号处理函数时有意义。
        SA_SIGINFO
            信号处理函数要使用3个参数而不是1个。也就是说应该设置sa_sigaction而不是sa_handler. 该标志只在建立信号处理函数时有意义。

### sighandler_t signal(int signum, sighandler_t handler);
#include <signal.h>  
typedef void (*sighandler_t)(int);  
在不同的UNIX版本间signal()的行为不同，并且历史上在Linux的不同版本间行为也不一样。
尽量避免使用，可以使用sigaction(2)代替。
 signal()设置处理信号码的处理器，可以是SIG_IGN（忽略）, SIG_DFL（默认行为）,或者是自定义的信号处理函数。 
 如果是自定义的信号处理函数，那么当进程收到信号时，要么处理函数被重置为SIG_DFL，要么信号被阻塞，然后会使用signum作为参数调用信号处理函数。
 如果信号因为处理函数的调用被阻塞了，那么在从信号处理函数返回后，信号会被解除阻塞。

 signal()返回之前的信号处理器的值，在出错时返回SIG_ERR.
 
 SIGKILL和SIGSTOP信号不能被忽略或者捕获。

### size_t fread(void *ptr, size_t size, size_t nmemb, FILE *stream);
#include <stdio.h>  
fread()从stream指向的流中读取nmemb个条目，每个条目是size个字节,然后将其存于ptr。
fread()不区分EOF和错误，调用方必须使用feof(3)和ferror(3)以决定发生了哪种情况.
       
### int atoi(const char *nptr);
#include <stdlib.h>
atoi()将nptr指针指向的字符串转换成整数。

###  int fileno(FILE *stream);
#include <stdio.h>  
fileno()检查参数stream并返回其底层的整数文件描述符。文件描述符仍由该流拥有，并会在调用fclose(3)时关闭。在传递给可能关闭文件描述符的代码之前使用dup(2)进行复制。

###  int fstat(int fd, struct stat *statbuf);
#include <sys/types.h>  
#include <sys/stat.h>  
#include <unistd.h>  
返回文件信息并将其设置到statbuf中。struct stat结构如下：

           struct stat {
               dev_t     st_dev;         /* ID of device containing file */
               ino_t     st_ino;         /* Inode number */
               mode_t    st_mode;        /* File type and mode */
               nlink_t   st_nlink;       /* Number of hard links */
               uid_t     st_uid;         /* User ID of owner */
               gid_t     st_gid;         /* Group ID of owner */
               dev_t     st_rdev;        /* Device ID (if special file) */
               off_t     st_size;        /* Total size, in bytes */
               blksize_t st_blksize;     /* Block size for filesystem I/O */
               blkcnt_t  st_blocks;      /* Number of 512B blocks allocated */

               /* 从Linux 2.6开始对下列时间戳字段，内核支持纳秒精度 */
               struct timespec st_atim;  /* Time of last access */
               struct timespec st_mtim;  /* Time of last modification */
               struct timespec st_ctim;  /* Time of last status change */
           };

### void rewind(FILE *stream);
#include <stdio.h>  
rewind()将流的文件位置指示符设置成文件开始。

### off_t ftello(FILE *stream);  long ftell(FILE *stream);
#include <stdio.h>  
ftell()获取流的文件位置指示符的当前值。ftello()的功能和ftell()是一样的，只是返回值类型是off_t。

### char *fgets(char *s, int size, FILE *stream);      
#include <stdio.h>  
fgets()最多读取从流中读取size-1个字符，并将其存储在s指向的buffer中。读取会在遇到EOF或者换行的时候停止。
如果读到了一个换行，则会将其存到buffer中。在最后一个字符后面会存储一个结束的null字节。  

### long int strtol(const char *nptr, char **endptr, int base);
#include <stdlib.h>  
将nptr指向的字符串转换成长整型，base代表的是数字的进制，必须在2到36之间，或者是特殊值0。
字符串可以以任意数量的空白字符开始，后面跟一个'+'或'-'。
如果base是0或16，字符串可以包含一个"0x"或者"0X"前缀，并且数字会作为16进制读取；
否则base为0会被视为10（10进制）除非字符串以'0'开始，这种情况下会被视为8进制。
剩下的字符串会被转换成一个长整型，转换会在发现第一个在指定base中无效的字符处停止（在base大于10时，'A'或者'a'代表10，'B'或者'b'代表11,以此类推，'Z'或者'z'代表35）

如果endptr不为NULL，那么第一个非法字符的地址会存储在 \*endptr。如果完全没有任何数字，会用\*endptr存储nptr并返回0。
返回时，如果*nptr不是'\0'，但是**endptr是'\0'，那么整个字符串都是合法的。

### int ioctl(int fd, unsigned long request, ...);
#include <sys/ioctl.h>  
ioctl()系统调用操作特殊文件的底层设备参数。特别的，很多字符特殊文件（例如终端）可以使用ioctl()控制。
参数fd必须是一个打开的文件描述符。
第二个参数一个和设备有关的请求码。
第三个参数传统上是char *argp(当时C语言中还没有void *)，后面会使用这一名称进行讨论。
一个ioctl()请求码将参数是输入还是输出及argp的大小都编码进请求码中。
用来指定一个ioctl()请求码的宏和定义都在<sys/ioctl.h>中。

如下代码会获取窗口大小：
    struct winsize ws;
    ioctl(1, TIOCGWINSZ, &ws)
winsize包含两个字段， ws.ws_col和ws.ws_row。

### int chdir(const char *path);
#include <unistd.h>
chdir()改变调用进程的当前工作目录。

###  pid_t fork(void);
#include <sys/types.h>
#include <unistd.h>

fork()通过复制调用进程来创建一个新进程。新的进程是作为原进程的子进程。
子进程和父进程运行在不同的内存空间。
在fork()的时候两者的内存空间有一样的内容。
一个进程的内存写，文件映射(mmap(2))和取消映射(munmap(2))不影响另一个进程。
除了一些属性，子进程是父进程的精确复制。见http://man7.org/linux/man-pages/man2/fork.2.html
fork()成功的话，在父进程里返回子进程的PID，在子进程里返回0。失败的话会返回-1，不会创建子进程。 

### setsid()
#include <sys/types.h>  
#include <unistd.h>  

如果调用进程不是进程组长那么setsid()会创建一个新会话。
调用进程是新的会话的组长（也就是说会话ID和进程ID一样）。调用线程也会变成会话中的线程组组长（也就是说线程ID和线程组ID一样）。
调用线程会变成新进程组和新会话中的唯一进程。
初始状态下，新会话没有控制终端。也就不会因为终端退出而退出。
可以参考credentials(7)看一个会话如何获取一个控制终端。

### int dup2(int oldfd, int newfd);
#include <unistd.h>  
dup2()系统调用创建oldfd的一个复制，也就是将oldfd复制到newfd。
如果newfd之前已经打开了，那么在重用之前该文件描述符会被默默关闭。
关闭和重用文件描述符newfd是一个原子操作。
这个很重要，因为如果试图使用close(2)和dup()可能易受竞态条件影响。

注意以下几点：
   - 如果oldfd不是一个合法的文件描述符，那么调用会失败，newfd不会被关闭。
   - 如果oldfd是一个合法文件描述符，newfd和oldfd有同样的值，那么dup2()什么都不做，返回newfd。

### long sysconf(int name);
#include <unistd.h>  

POSIX允许应用在编译时或运行时测试特定选项是否支持及它们的值。
在编译时这是由包含<unistd.h>和<limits.h>，然后测试特定的宏的值来实现的。
在运行时，可以用sysconf()来获取数值；可以使用fpathconf(3)和pathconf(3)来获取文件系统相关数值； 
可以使用confstr(3)来获取字符串值。
这些函数获得的值都是系统配置常量，不会在进程的声明周期内改变。
   - _SC_PHYS_PAGES
         物理内存的页数。注意，该值和_SC_PAGESIZE的乘积可能溢出。
   - PAGESIZE - _SC_PAGESIZE
         以字节为单位的页的大小。

### int unlink(const char *pathname);
#include <unistd.h>
unlink()从文件系统删除一个名称。
如果这个名称是一个文件的最后一个链接，并且在任何进程中文件都不处于打开状态，那么这个文件会被删除，它用的空间会被回收。
如果这个名称是一个文件的最后一个链接，但是在有些进程中文件处于打开状态，那么这个文件仍会存在直到指向其的最后一个文件描述符被关闭。
如果名称指向一个符号链接，该链接会被删除。
如果名称指向一个socket，FIFO或device，那么名称会被删除，但是打开了这些对象的进程仍然可以继续使用它。

### int getaddrinfo(const char *node, const char *service,const struct addrinfo *hints,struct addrinfo **res);
#include <sys/types.h>  
#include <sys/socket.h>  
#include <netdb.h>  

给定node和service（也就是指定一个Internet主机和服务），getaddrinfo()返回一个或多个addrinfo结构，
每个结构包含一个可以用于调用bind(2)或者connect(2)的网络地址。
getaddrinfo()函数组合了gethostbyname(3)和getservbyname(3)的功能，但是和后者不同，getaddrinfo()是可重入的，并且去除了对IPV4还是IPV6的依赖。

addrinfo结构体包含以下字段：
           struct addrinfo {
               int              ai_flags;
               int              ai_family;
               int              ai_socktype;
               int              ai_protocol;
               socklen_t        ai_addrlen;
               struct sockaddr *ai_addr;
               char            *ai_canonname;
               struct addrinfo *ai_next;
           };

hints参数指定了选择由res返回的socket地址结构体的标准。
如果hints不是NULL，那么它指向的addrinfo结构体的ai_family, ai_socktype和ai_protocol字段指定了限制由getaddrinfo()返回的socket地址集合的标准。
      - ai_family 该字段指定了想要的地址族。合法的值有AF_INET和AF_INET6. AF_UNSPEC表明应该返回可用于node和service的任意地址族。

      - ai_socktype 该字段指定了想要的socket类型，例如SOCK_STREAM和SOCK_DGRAM.指定0表明可以返回任意类型。

      - ai_protocol 该字段指定了返回rocket地址的协议。指定0表示可以返回任意协议。

      - ai_flags  该字段指定了额外的选项。多个选项之间通过OR组合在一起。

hints中的其他字段必须设置为0或者NULL指针。
指定hints为NULL等价于设置ai_socktype和ai_protocol为0;设置ai_family为AF_UNSPEC;设置ai_flags为(AI_V4MAPPED | AI_ADDRCONFIG).  (POSIX为ai_flags指定了不同的默认值).

node要么指定一个数组网络地址（对IPV4，像inet_aton(3)支持的点分表示法;对IPv6,像inet_pton(3)支持的16进制字符串),或者一个网络主机名，其地址会由系统查询和解析。

如果hints.ai_flags包含AI_NUMERICHOST标记，node必须是一个数值的网络地址。AI_NUMERICHOST标记抑制了所有潜在的网络主机地址查找。

如果hints.ai_flags包含AI_PASSIVE标记，并且node为NULL，那么返回的socket地址将会适合于bind(2)一个会accept(2)连接的socket.
返回的地址会包含通配地址(对IPV4是INADDR_ANY，对IPV6是IN6ADDR_ANY_INIT).通配地址用于想在主机的任一网络地址上接受连接的应用（通常是服务器）。
如果node不为NULL，那么AI_PASSIVE标记会被忽略.

如果hints.ai_flags不包含AI_PASSIVE标记，那么返回的socket地址适用于connect(2),sendto(2), 和endmsg(2)。
如果node为NULL那么网络地址会被设置为回环地址(对IPV4为INADDR_LOOPBACK, 对IPV6是IN6ADDR_LOOPBACK_INIT); 这可以用于想和同一台主机上的对端通信的应用。

service设置每个返回的地址结构体的端口。
如果该参数是一个服务名(见services(5)), 它会被翻译为对应的端口号。
该参数也可以设置为一个十进制数，该数会直接转换成数值。
如果service是NULL，那么端口号就是未初始化的。
如果hints.ai包含了AI_NUMERICSERV标记，那么服务必须指向一个包含数值端口号的字符串。该标记会阻止对名称解析服务的调用。
node或者service都可以为NULL，但是两者不能同时为NULL。

getaddrinfo()分配和初始化一个addrinfo结构体的链表。每一个结构体都指向和node和service匹配并且受限于hints的网络地址。
getaddrinfo()会返回列表的起始指针给res。列表项由ai_next字段链接起来。

通常引用应该试着按照地址返回的顺序使用他们。
getaddrinfo()使用的排序函数定义在RFC 3484;可以通过编辑/etc/gai.conf (从glibc 2.5可用)来调整顺序.

### int bind(int sockfd, const struct sockaddr *addr, socklen_t addrlen);
#include <sys/types.h>  
#include <sys/socket.h>  
当一个socket由socket(2)创建后,它存在于一个命名空间（地址族），但是没有任何地址分配给它。
bind()通过指定addr，将文件描述符sockfd和addr绑定。
addrlen以字节为单位，指定address结构体的大小。传统上，该操作叫做分配一个名称给一个socket。
通常在一个SOCK_STREAM socket可以接受连接前通过bind()分配一个本机地址是必须的。

不同的地址族名称绑定规则并不相同，请查看相应手册。AF_INET, 见ip(7); AF_INET6, 见ipv6(7); AF_UNIX,见packet(7);
addr参数的实际结构取决于地址族。
      sockaddr的结构定义类似于下:

           struct sockaddr {
               sa_family_t sa_family;
               char        sa_data[14];
           }
      该结构的唯一目的是用来强制转换addr结构指针以避免编译器警告。

### int listen(int sockfd, int backlog);
#include <sys/types.h>
#include <sys/socket.h>

listen()标记由sockfd指向的socket为被动socket，也就是一个使用accept(2)来接受进来的连接的socket.
sockfd参数是一个指向SOCK_STREAM或者SOCK_SEQPACKET类型socket的文件描述符。

backlong参数定义了在sockfd上等待的连接队列的最大长度。
如果在队列满了时，来了一个连接请求，那么客户端可能收到一个ECONNREFUSED错误，或者如果底层协议支持重传，请求可能会被忽略这样之后的重试可能会成功。

###   int setitimer(int which, const struct itimerval *new_value, struct itimerval *old_value);

#include <sys/time.h>

该系统调用提供了访问间隔定时器（也就是一开始在未来某个时间点过期，并且可选地在那之后周期性的过期）的方法。
当一个定时器过期，会为调用进程产生一个信号，然后如果间隔不为0，那么定时器会重置为该间隔。

不同的时钟在定时器过期时会产生不同的信号，类型由which参数指定：

       - ITIMER_REAL  该定时器使用真实时间(挂钟)倒数,会产生一个SIGALRM信号。

       - ITIMER_VIRTUAL 该定时器使用由进程消费的用户态CPU时间（包括进程中的所有线程）倒数。在过期时，产生一个SIGVTALRM信号。

       - ITIMER_PROF 该定时器使用由进程消费的总的（包括用户时间和系统时间）CPU时间（包括进程中的所有线程）倒数。在过期时，产生一个SIGPROF信号.

一个进程只能有这三种类型中的一种定时器。
定时器值由以下结构体定义：

           struct itimerval {
               struct timeval it_interval; /* Interval for periodic timer，周期性间隔 */
               struct timeval it_value;    /* Time until next expiration 下次过期时间*/
           };

           struct timeval {
               time_t      tv_sec;         /* seconds */
               suseconds_t tv_usec;        /* microseconds */
           };

setitimer()函数通过设置定时器类型为which，值为new_value来设置或取消定时器。
如果old_value不为NULL，那么old_value会存储定时器原先的值。
如果new_value.it_value的任一字段不为0，那么定时器会在指定时间过期。如果两者都为0，那么定时器会被取消。
new_value.it_interval字段指定新的定时器间隔，如果两个子字段都为0，那么该定时器就只会执行一次。
              
###  struct tm *localtime(const time_t *timep); struct tm *localtime_r(const time_t *timep, struct tm *result);
#include <time.h>   
localtime()函数将日历时间timep转换为拆分的时间表示，拆分后的时间表示是和用户的时区关联的。
该函数表现的像是先调用tzset(3)，然后使用当前时区设置外部变量tzname；
使用以秒为单位的UTC时间和当前时间的差值设置timezone字段；
如果需要应用夏令时规则，设置daylight为非0.
返回值执行一个静态分配的结构体，该结构体可能会被之后的日期和时间函数覆盖。
localtime_r()函数的功能是一样的，但是将值存到一个用户提供的结构体中。
不需要设置tzname，timezone和daylight。

### int kill(pid_t pid, int sig);
#include <sys/types.h>  
#include <signal.h>  
kill()系统调用用于给任意进程或进程组发送信号。
如果pid是正数，信号sig会发送给由pid指定的进程。
如果pid为0，sig会被发送给调用进程所在的进程组中的所有进程。
如果pid为-1，sig会被发送给调用进程有权限发送信号的所有进程，除了进程1（init）。
如果pid小于-1，sig会被发送给ID为-pid的进程组内的所有进程。

如果sig为0，那么不会发送信号，但是仍然会进行进程存在和权限检查；这可以用于检查调用者有权限发送信号的进程ID或者进程组ID是否存在。

一个进程要想有权限发送信号，它必须要么有权限（在Linux下，在目标进程的用户空间有CAP_KILL能力），
或者发送进程的real或者effective用户ID和目标进程的real或者saved set-user-ID一样。
对于SIGCONT，发送和接收进程属于同一个会话就足够了。

### pid_t wait3(int *wstatus, int options,struct rusage *rusage);
#include <sys/types.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <sys/wait.h>

该函数是非标准的；在新程序中，最好使用waitpid(2)或者waitid(2).
wait3()系统调用和waitpid(2)相同，只是会通过rusage来来返回资源使用信息。
wait系列系统调用用来等待调用进程的子进程的状态改变，并且获取子进程的相关信息。
一个状态改变包括：子进程终止；子进程被一个信号停止；或者子进程被一个信号恢复。
在子进程终止的情况下，执行wait允许系统释放子进程关联的资源；如果没有执行wait，那么结束的子进程处于僵尸状态。
如果子进程已经改变了状态，那么这些调用会立即返回。否则，它们会阻塞直到子进程改变状态，或者一个信号处理器中断了该调用。

除了rusage参数的使用，下列调用是等同的：
    wait3(wstatus, options, rusage);
    和
    waitpid(-1, wstatus, options);

    wait4(pid, wstatus, options, rusage);
    和
    waitpid(pid, wstatus, options);
也就是说，wait3()等待任一子进程，wait4()可以指定要等待的子进程。

waitpid()系统调用挂起调用线程的执行直到pid引用的一个子进程发生状态改变。
默认情况下，waitpid()只等待终止了的children，但是这一行为可以被下面描述的options参数修改。 waits only for terminated children, but this
    pid的值可以为：

    -   < -1 表示等待和pid的绝对值相同的进程组的任一子进程。

    -   -1   表示等待任一子进程。

    -   0   表示等待和当前调用进程的进程组下面的任一子进程。

    -   > 0  表示等待进程ID为pid的子进程。

     options的值由以下常量OR得来：

       WNOHANG     如果没有任何子进程退出，那么立即返回。

如果rusage不为NULL，那么rusage结构体就记录子进程的一些资源使用情况，主要是内存使用情况。

可以使用下列宏来获取状态：
    - WIFEXITED(wstatus)
        如果子进程正常终止，那么返回true，也就是调用exit(3)或者_exit(2),或者从main()返回.
    - WEXITSTATUS(wstatus)
        返回子进程的退出状态。包括子进程在调用exit(3)或者_exit(2)或者main()函数中指定的return语句时指定的最少8比特有效位数的状态参数。
        该宏只能在WIFEXITED为true的时候调用。
    - WIFSIGNALED(wstatus)
        如果子进程被一个信号终止返回true。
    - WTERMSIG(wstatus)
        返回造成子进程终止的信号。该宏应该只在WIFSIGNALED返回true时使用。

### int accept(int sockfd, struct sockaddr *addr, socklen_t *addrlen);
#include <sys/types.h>  
#include <sys/socket.h>  
accept()系统调用用于基于连接的socket类型(SOCK_STREAM, SOCK_SEQPACKET).
它从监听socket sockfd的等待连接请求队列中取出第一个请求，创建一个连接了的socket，并将引用该socket的文件描述符返回。
新创建的socket不处于监听状态，原来的socket sockfd不会该调用影响。

参数sockfd是一个使用socket(2)创建，使用bind(2)绑定到本地地址，并且在调用listen(2)之后正在监听连接的一个socket。

参数addr是一个指向sockaddr结构的指针。该结构会设置为对端socket的地址。
具体的格式取决于socket的地址族。如果addr为NULL，那么不会设置该信息，同时addrlen也未使用，并且同样应该设置为NULL。

addrlen参数是一个值结果参数：调用方必须初始化它以包含addr结构体的大小；在返回时，它会包含对端地址的实际大小。
如果提供的缓冲太小，那么返回的地址会被截断；在这种情况下，addrlen会返回一个比其初始值更大的值。

如果队列中没有任何挂起的连接请求，并且socket没有标记为非阻塞，那么accept()会阻塞调用者，直到存在一个连接请求。
如果socket标记为非阻塞，并且队列中没有任何挂起的连接请求,accept()以错误EAGAIN或者EWOULDBLOCK失败。

为了获取一个socket上进来的连接的通知，可以使用select(2), poll(2),或者epoll(7).
当一个新连接尝试连接时，会发送一个可读事件，这时就可以调用accept()来为那个连接获取一个socket。另外，作为替代，可以设置当socket上产生活动时，发送SIGIO信号。

### int epoll_wait(int epfd, struct epoll_event *events,int maxevents, int timeout);
#include <sys/epoll.h>  
epoll_wait()系统调用等待文件描述符epfd引用的epoll实例上的事件。
events参数会包含调用者可用的事件。最多可用返回maxevents个事件。
timeout参数指定了epoll_wait()会阻塞的毫秒数。时间是使用CLOCK_MONOTONIC（单调时钟）来计量的。

调用会一直阻塞，直到满足以下条件之一：

       *  一个文件描述符发送一个事件;

       *  调用被一个信号处理器中断;

       *  超时时间过期了.
注意，超时时间会按照系统时钟粒度取整，并且内核调度延时意味着阻塞时间可能会超过一点。
指定超时时间为-1会导致epoll_wait()一直阻塞；指定超时时间为0，会导致epoll_wait()立即返回，即使没有任何事件可用。

调用成功时epoll_wait()返回可以接收请求的I/O操作的文件描述符的数量，或者在超时时间内，如果没有文件描述符就绪，则返回0.
如果发生错误，epoll_wait()返回-1，并且设置相应的errno.

因为就绪的事件在内核中是由一个链表维护的。

使用Linux epoll模型，水平触发模式；当socket可写时，会不停的触发 socket 可写的事件，如何处理？

处理方式：

开始不把 socket 加入 epoll，需要向 socket 写数据的时候，直接调用 write 或者 send 发送数据。
如果返回 EAGAIN，把 socket 加入 epoll，在 epoll 的驱动下写数据，全部数据发送完毕后，再移出 epoll。

### int sched_setaffinity(pid_t pid, size_t cpusetsize,const cpu_set_t *mask);int sched_getaffinity(pid_t pid, size_t cpusetsize, cpu_set_t *mask);

#define _GNU_SOURCE  
#include <sched.h>  
一个线程的CPU亲和掩码决定了线程想在其上运行的CPU集合。
在多处理器系统上，设置CPU亲和掩码可以用来提高性能。
例如，通过将一个CPU指定给一个特定线程，可能可以保证该线程的最大执行速度。
将一个线程限制到一个CPU上统一避免了因为线程在不同CPU之间切换运行导致的缓存失效造成的性能损耗。
CPU亲和掩码由cpu_set_t结构体类型的参数mask表示。有一些操作CPU集合的宏请见CPU_SET(3).

sched_setaffinity()设置ID为pid的线程的CPU亲和掩码为mask。
如果pid为0，那么使用调用线程。
参数cpusetsize是mask指向的数据的长度。通常该参数由sizeof(cpu_set_t)指定.

如果pid指定的线程当前并没有运行在由mask指定的CPU上，那么线程会被迁移到mask指定的CPU集合中的其中一个CPU上。

### int pthread_setcancelstate(int state, int *oldstate); int pthread_setcanceltype(int type, int *oldtype);
#include <pthread.h>  
pthread_setcancelstate()设置调用线程的可取消性状态。state参数必须为以下值之一：

    - PTHREAD_CANCEL_ENABLE
        线程可取消。这是所有的新线程的默认可取消性状态。线程的可取消性状态决定了线程如何响应一个取消请求。

    - PTHREAD_CANCEL_DISABLE
        线程不可取消。如果收到一个取消请求，那么它会一直阻塞直到开启可取消性。

pthread_setcanceltype()设置调用线程的可取消性类型。type参数必须为以下值之一s:

    - PTHREAD_CANCEL_DEFERRED
        一个取消请求被推迟直到线程下次调用一个是取消点的函数(见pthreads(7))。
        这是所有新线程的默认取消类型。

    - PTHREAD_CANCEL_ASYNCHRONOUS
        线程可以在任意时间取消。（典型情况下，线程会在收到取消请求后立即取消，但是系统不保证这一点。）


### int pthread_cond_broadcast(pthread_cond_t *cond); int pthread_cond_signal(pthread_cond_t *cond);
#include <pthread.h>

pthread_cond_broadcast()解除所有当前阻塞在条件变量cond上的线程。
pthread_cond_signal()解除至少一个（如果有的话）当前阻塞在条件变量cond上的线程。

如果超过一个线程阻塞在条件变量上，调度策略应该决定线程解除阻塞的顺序。
当线程解除阻塞时，线程拥有当它调用pthread_cond_wait()时的mutex。

pthread_cond_broadcast()或者pthread_cond_signal()可以从一个线程调用，不管其当前是否拥有调用pthread_cond_wait()时的mutex。
然而如果可预见的调度行为是必要的，那么调用pthread_cond_broadcast()或者pthread_cond_signal()的线程应该先锁住对应的mutex.

在没有任何线程阻塞在条件上时，pthread_cond_broadcast()和pthread_cond_signal()应该没有作用。

### void *dlopen(const char *filename, int flags);
#include <dlfcn.h>  
dlopen()加载由null结尾的字符串filename指定的动态共享对象文件，并且返回一个句柄。
该句柄会用于dlopen API中的其他函数，例如dlsym(3),dladdr(3), dlinfo(3), and dlclose().  
如果filename为NULL，那么返回的句柄代表了main程序。如果filename包含了一个斜线("/"),那么它会被解析为一个路径(相对或绝对路径)。
否则动态连接器按照如下顺序搜索对象(详见ld.so(8)):

       o   (ELF only)如果调用程序的可执行文件(注意这个是指加载动态对象的程序而不是动态对象文件)包含一个DT_RPATH标签，并且不包含DT_RUNPATH标签,那么会搜索DT_RPATH标记中列的目录。

       o   如果在程序启动的时候环境变量LD_LIBRARY_PATH设置为一个分号分隔的目录列表，那么会搜索这些列表（考虑到安全， set-user-ID和set-group-ID程序会忽略该变量）。

       o   (ELF only) 如果程序的可执行文件包含一个DT_RUNPATH标签，会搜索该标签列的目录列表。

       o   检查缓存文件/etc/ld.so.cache (由ldconfig(8)维护)看其是否包含文件名入口。

       o   按顺序搜索/lib和/usr/lib目录。

       如果由文件名指定的对象依赖其他共享对象，那么动态链接器会使用同样的规则加载它们。

       flags必须包含以下两个值中的一个：

       RTLD_LAZY
              执行懒绑定。只在引用它们的代码执行的时候才解析符号。如果symbol从来没被引用过，那么永远也不会解析。(懒绑定只用于函数引用；变量总是在共享对象加载的时候立即绑定）。
              从glibc 2.1.1开始,该标记的作用会被LD_BIND_NOW环境变量覆盖.

       RTLD_NOW
              如果指定了该值，或者环境变量LD_BIND_NOW设置为一个非空字符串，在dlopen()返回前，会解析共享对象中所有未定义的符号。如果失败，会返回一个错误。

       零活多个以下值可以通过OR的方式来设置到flags中：

       RTLD_GLOBAL
              该共享对象定义的符号对后续加载的共享对象可见。

       RTLD_LOCAL
              和RTLD_GLOBAL的作用相反,如果两个标记都没设置，那么默认是RTLD_LOCAL。

       RTLD_NODELETE (since glibc 2.2)
              dlclose()的时候不卸载共享对象.所以，如果之后使用dlopen()重加载，对象的静态和全局变量不会被重新初始化。

       RTLD_NOLOAD (since glibc 2.2)
              不加载共享对象。可以用于测试对象是否已经存在。该标记还可以用于提升已经加载的共享对象的标记。例如，之前使用RTLD_LOCAL加载的共享对象可以使用RTLD_NOLOAD | RTLD_GLOBAL重新打开。

       RTLD_DEEPBIND (since glibc 2.3.4)
              设置该共享对象符号的查找范围比全局范围靠前。这意味着一个自包含的对象会优先使用它自己的符号，而不是已经加载了的有着相同名字的全局符号。

       如果filename为NULL，那么返回的句柄代表主程序。当传递给 dlsym()，该句柄造成在主程序中搜索符号，然后是所有由使用RTLD_GLOBAL标记的dlopen()加载的共享对象。

       共享对象的符号引用按以下顺序解析：
       主程序及其依赖加载的对象列表；由使用RTLD_GLOBAL标记的dlopen()加载的共享对象及其依赖；共享对象自身的定义(及为了该对象加载的所有依赖).

       由ld(1)载入到可执行对象中的动态符号表的任意全局符号，也可以用于解析动态加载共享对象引用。
       符号可能是由可执行对象通过"-rdynamic" (或"--export-dynamic")标记链接时加入到动态符号表，这种方式会导致可执行文件的所有全局符号加入到动态符号表；
       或者是ld(1)在静态链接时注意到了对另一个对象的符号的依赖。

       如果使用dlopen()再一次打开相同的共享对象，会返回相同的句柄。动态链接器维护着对象句柄的引用计数，所以一个动态加载共享对象直到dlclose()的调用次数和dlopen()一样的时候，才会回收。
       构造器只有在对象实际加载进内存的时候才会调用（也就是引用计数增长到1的时候）。

       一个使用RTLD_NOW的后续dlopen()调用可能会强制一个之前使用RTLD_LAZY加载的共享对象进行符号解析。类似地，一个之前使用RTLD_LOCAL打开的对象可以被后续的dlopen()提升为RTLD_GLOBAL。

### void *dlsym(void *handle, const char *symbol);void *dlvsym(void *handle, char *symbol, char *version);
#include <dlfcn.h>  
在一个共享对象或者可执行对象中获取一个符号的地址。
dlsym()拿着一个由dlopen(3)打开的动态加载共享对象的"handle"和一个由null结尾的symbol名称，然后返回symbol载入的内存地址。
