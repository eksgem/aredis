## 内存分配
redis可以根据配置使用不同的内存分配库。主要有zmalloc.h和zmalloc.c实现。  
目前支持的内存分配库有tcmalloc,jemalloc,苹果的c库，glibc,标准libc等。

    #if defined(USE_TCMALLOC)
    #define ZMALLOC_LIB ("tcmalloc-" __xstr(TC_VERSION_MAJOR) "." __xstr(TC_VERSION_MINOR))
    #include <google/tcmalloc.h>
    #if (TC_VERSION_MAJOR == 1 && TC_VERSION_MINOR >= 6) || (TC_VERSION_MAJOR > 1)
    #define HAVE_MALLOC_SIZE 1
    #define zmalloc_size(p) tc_malloc_size(p)
    #else
    #error "Newer version of tcmalloc required"
    #endif

    #elif defined(USE_JEMALLOC)
    #define ZMALLOC_LIB ("jemalloc-" __xstr(JEMALLOC_VERSION_MAJOR) "." __xstr(JEMALLOC_VERSION_MINOR) "." __xstr(JEMALLOC_VERSION_BUGFIX))
    #include <jemalloc/jemalloc.h>
    #if (JEMALLOC_VERSION_MAJOR == 2 && JEMALLOC_VERSION_MINOR >= 1) || (JEMALLOC_VERSION_MAJOR > 2)
    #define HAVE_MALLOC_SIZE 1
    #define zmalloc_size(p) je_malloc_usable_size(p)
    #else
    #error "Newer version of jemalloc required"
    #endif

    #elif defined(__APPLE__)
    #include <malloc/malloc.h>
    #define HAVE_MALLOC_SIZE 1
    #define zmalloc_size(p) malloc_size(p)
    #endif

    #ifndef ZMALLOC_LIB
    #define ZMALLOC_LIB "libc"
    #ifdef __GLIBC__
    #include <malloc.h>
    #define HAVE_MALLOC_SIZE 1
    #define zmalloc_size(p) malloc_usable_size(p)
    #endif
    #endif

    /* We can enable the Redis defrag capabilities only if we are using Jemalloc
    * and the version used is our special version modified for Redis having
    * the ability to return per-allocation fragmentation hints. */
    #if defined(USE_JEMALLOC) && defined(JEMALLOC_FRAG_HINT)
    #define HAVE_DEFRAG
    #endif

内存分配也并不是直接使用zmalloc,而是使用s_malloc

    sh = s_malloc(hdrlen+initlen+1);
可以通过改变宏定义，来使用自己需要的其他内存分配库，该宏定义在sdsalloc.h。

    #define s_malloc zmalloc      
使用的malloc也并不是标准库函数，而是定义成一个宏。根据配置展开成不同的函数调用。

    /* Explicitly override malloc/free etc when using tcmalloc. */
    #if defined(USE_TCMALLOC)
    #define malloc(size) tc_malloc(size)
    #define calloc(count,size) tc_calloc(count,size)
    #define realloc(ptr,size) tc_realloc(ptr,size)
    #define free(ptr) tc_free(ptr)
    #elif defined(USE_JEMALLOC)
    #define malloc(size) je_malloc(size)
    #define calloc(count,size) je_calloc(count,size)
    #define realloc(ptr,size) je_realloc(ptr,size)
    #define free(ptr) je_free(ptr)
    #define mallocx(size,flags) je_mallocx(size,flags)
    #define dallocx(ptr,flags) je_dallocx(ptr,flags)
    #endif
    #endif

当前的内存分配由zmalloc完成，可以通过分配的内存指针，来获得分配的内存的大小。上面提到的那些库中，除了标准库，其他库都有通过分配的内存指针获得内存大小的能力，如果是这些支持的库，则会定义一个宏HAVE_MALLOC_SIZE来标记。而对于不支持这一能力的情况，redis自己进行了处理。  
首先是定义一个宏PREFIX_SIZE，该宏用来表示记录分配内存的大小需要使用几个字节。如果定义了HAVE_MALLOC_SIZE宏，则PREFIX_SIZE就是0，因为不需要redis自己实现，库函数已经实现了，否则，redis就会自己实现这一功能。

    #ifdef HAVE_MALLOC_SIZE
    #define PREFIX_SIZE (0)
    #else
    #if defined(__sun) || defined(__sparc) || defined(__sparc__)
    在sun系统上，为sizeof(long long)，
    #define PREFIX_SIZE (sizeof(long long))
    #else
    否则为sizeof(size_t)
    #define PREFIX_SIZE (sizeof(size_t))
    #endif
    #endif
下面我们看下zmalloc的代码：
    void *zmalloc(size_t size) {
        void *ptr = malloc(size+PREFIX_SIZE);

        if (!ptr) zmalloc_oom_handler(size);
    如果定义了HAVE_MALLOC_SIZE，则分配的内存的指针和大小都由库函数来处理。
    #ifdef HAVE_MALLOC_SIZE
        这里会记录已经分配了的总内存的大小。
        update_zmalloc_stat_alloc(zmalloc_size(ptr));
        return ptr;
    如果没有定义HAVE_MALLOC_SIZE，则redis自己实现了该功能。在分配内存的时候，在起始位置多分配了一块记录内存大小的长度为PREFIX_SIZE的内存。这样分配的内存的大小就为PREFIX_SIZE+需要分配的实际内存的大小，而返回的指针为实际分配的指针(需要转型为char*以进行正确的指针运算)+PREFIX_SIZE。
    #else
        *((size_t*)ptr) = size;
        update_zmalloc_stat_alloc(size+PREFIX_SIZE);
        return (char*)ptr+PREFIX_SIZE;
    #endif
    }
记录总的分配的内存的大小，

    #define update_zmalloc_stat_alloc(__n) do { \
        size_t _n = (__n); \
        //假设底层分配内存是以long的大小的整数倍分配的
        if (_n&(sizeof(long)-1)) _n += sizeof(long)-(_n&(sizeof(long)-1)); \
        atomicIncr(used_memory,__n); \
    } while(0)        
其中zmalloc_size也是根据内存分配库的不同而有不同的实现：

    #if defined(USE_TCMALLOC)
    #define zmalloc_size(p) tc_malloc_size(p)

    #elif defined(USE_JEMALLOC)
    #define zmalloc_size(p) je_malloc_usable_size(p)

    #elif defined(__APPLE__)
    #define zmalloc_size(p) malloc_size(p)

    #ifdef __GLIBC__
    #define zmalloc_size(p) malloc_usable_size(p)
    //如果库函数没有定义，则使用自己定义的：
    /* Provide zmalloc_size() for systems where this function is not provided by
    * malloc itself, given that in that case we store a header with this
    * information as the first bytes of every allocation. */
    #ifndef HAVE_MALLOC_SIZE
    size_t zmalloc_size(void *ptr) {
        void *realptr = (char*)ptr-PREFIX_SIZE;
        size_t size = *((size_t*)realptr);
        /* Assume at least that all the allocations are padded at sizeof(long) by
        * the underlying allocator. */
        //假设分配的内存是long的大小的整数倍，这里size主要是用于记录内存分配情况，并不会用在释放内存等方面，所以这里如果不是那么精确没有影响。       
        if (size&(sizeof(long)-1)) size += sizeof(long)-(size&(sizeof(long)-1));
        return size+PREFIX_SIZE;
    }   
zcalloc和zrealloc也是类似的实现方法。