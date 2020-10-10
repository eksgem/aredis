## redis的动态字符串(sns)实现
sns是Simple Dynamic Strings的简写，是二进制安全的。由sds.h和sds.c实现。
### sds.h
除了一些函数声明外，sds.h主要定义了几种sns header结构。
sds是一个char*。

    typedef char *sds;
下面这几个是几种不同的snshdr结构定义，主要由字符串长度决定使用哪一种,这样可以根据不同的长度使用不同的结构以节省内存。 __attribute__ ((__packed__))是告诉编译器取消结构在编译过程中的优化对齐,这样可以保证各字段在地址上是紧邻在一起的，是GCC特有的语法.这样才能通过结构内的指针偏移借由sns找到其相邻的flags字段。另外，这里结构已经考虑过对齐优化了，像是前面放的都是sizeof较大的类型，并且都是对齐的，比如int都在4字节处对齐，short都在2字节处对齐。
当字符串长度小于1<<5时，使用sdshdr5.由注释可知，没有单独的长度字段，而是由flags中拿出高5位来作为长度，低3位用来表示header的类型。

    /* Note: sdshdr5 is never used, we just access the flags byte directly.
    * However is here to document the layout of type 5 SDS strings. */
    struct __attribute__ ((__packed__)) sdshdr5 {
        unsigned char flags; /* 3 lsb of type, and 5 msb of string length */
        char buf[];
    };
header类型的定义如下，判断的时候由SDS_TYPE_MASK和类型进行按位与即可得到实际类型：

    #define SDS_TYPE_5  0
    #define SDS_TYPE_8  1
    #define SDS_TYPE_16 2
    #define SDS_TYPE_32 3
    #define SDS_TYPE_64 4
    #define SDS_TYPE_MASK 7
    #define SDS_TYPE_BITS 3    
当字符串长度小于1<<8时，使用sdshdr8. 有一个len字段，表示字符串的真正长度，不包含终止空字符；一个alloc字段，表示sns的最大容量，不包含header和空结束符，一个flags字段,低三位用来表示类型，和一个char buf[],表明实际存储的字符数组。后面sdshdr16，sdshdr32，sdshdr64都是类似的。

    struct __attribute__ ((__packed__)) sdshdr8 {
        uint8_t len; /* used */
        uint8_t alloc; /* excluding the header and null terminator */
        unsigned char flags; /* 3 lsb of type, 5 unused bits */
        char buf[];
    };
    struct __attribute__ ((__packed__)) sdshdr16 {
        uint16_t len; /* used */
        uint16_t alloc; /* excluding the header and null terminator */
        unsigned char flags; /* 3 lsb of type, 5 unused bits */
        char buf[];
    };
    struct __attribute__ ((__packed__)) sdshdr32 {
        uint32_t len; /* used */
        uint32_t alloc; /* excluding the header and null terminator */
        unsigned char flags; /* 3 lsb of type, 5 unused bits */
        char buf[];
    };
    struct __attribute__ ((__packed__)) sdshdr64 {
        uint64_t len; /* used */
        uint64_t alloc; /* excluding the header and null terminator */
        unsigned char flags; /* 3 lsb of type, 5 unused bits */
        char buf[];
    };  
下面是几个宏定义和内联函数定义：  
##是将两个符号连接成一个，如sdshdr和8（T为8）合成sdshdr8，这里T代表的是header的类型，s是一个sds。  
获取header结构的指针变量，后续可以通过sh这个变量来引用这个指针。这个找header指针的过程为，传递的s为一个sds，也就是header结构的最后一个字段，并且其类型为char*，根据指针算法，char*类型的指针-1就是指针往前移动1，这里有一个关键的一点是：char buf[]是一个柔性数组，所以并不占用header结构体的大小，所以sizeof(struct sdshdr##T)返回的值中并不包括char buf[]的大小，这样才可以用下面的方式得到header。如果不使用柔性数组，那么应该使用char *来声明，下面的减法中应该再减去8，因为char\*占用了8字节，当然指针大小在不同系统可能是不同的。同样我们也可以知道，数组和指针在很多地方都是不一样的，比如在结构体中占用的大小，如果是普通数组，那么就应该是加上普通数组的大小，而如果是指针，那么就是一个指针的大小。

    #define SDS_HDR_VAR(T,s) struct sdshdr##T *sh = (void*)((s)-(sizeof(struct sdshdr##T)));
找到并强制转换为header结构指针

    #define SDS_HDR(T,s) ((struct sdshdr##T *)((s)-(sizeof(struct sdshdr##T))))
获取sdshdr5的长度

    #define SDS_TYPE_5_LEN(f) ((f)>>SDS_TYPE_BITS)
获取字符串长度，从上面的各种定义我们可以知道，在各种函数中传递的并不是上面的各种header结构指针，而是传递sds，一是sds可以直接当成普通C字符串使用，另外，需要通过sds来找到其所在的header结构，因为各个header结构并不一样。这样后面的各种操作就比较明显了。

    static inline size_t sdslen(const sds s) {
        // 通过s[-1]可以找到flags。这样也就要求各个snshdr的字段必须是有顺序的，即char buf[]紧邻在flags之后。注意这里的指针算法，s[-1]并不是s-1，而是*(s-1),得到的是指针指向的具体值。
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5:
                return SDS_TYPE_5_LEN(flags);
            case SDS_TYPE_8:
                return SDS_HDR(8,s)->len;
            case SDS_TYPE_16:
                return SDS_HDR(16,s)->len;
            case SDS_TYPE_32:
                return SDS_HDR(32,s)->len;
            case SDS_TYPE_64:
                return SDS_HDR(64,s)->len;
        }
        return 0;
    }
可用大小等于容量-已用大小
    static inline size_t sdsavail(const sds s) {
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5: {
                return 0;
            }
            case SDS_TYPE_8: {
                SDS_HDR_VAR(8,s);
                return sh->alloc - sh->len;
            }
            case SDS_TYPE_16: {
                SDS_HDR_VAR(16,s);
                return sh->alloc - sh->len;
            }
            case SDS_TYPE_32: {
                SDS_HDR_VAR(32,s);
                return sh->alloc - sh->len;
            }
            case SDS_TYPE_64: {
                SDS_HDR_VAR(64,s);
                return sh->alloc - sh->len;
            }
        }
        return 0;
    }

    static inline void sdssetlen(sds s, size_t newlen) {
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5:
                {
                    unsigned char *fp = ((unsigned char*)s)-1;
                    *fp = SDS_TYPE_5 | (newlen << SDS_TYPE_BITS);
                }
                break;
            case SDS_TYPE_8:
                SDS_HDR(8,s)->len = newlen;
                break;
            case SDS_TYPE_16:
                SDS_HDR(16,s)->len = newlen;
                break;
            case SDS_TYPE_32:
                SDS_HDR(32,s)->len = newlen;
                break;
            case SDS_TYPE_64:
                SDS_HDR(64,s)->len = newlen;
                break;
        }
    }

    static inline void sdsinclen(sds s, size_t inc) {
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5:
                {
                    unsigned char *fp = ((unsigned char*)s)-1;
                    unsigned char newlen = SDS_TYPE_5_LEN(flags)+inc;
                    *fp = SDS_TYPE_5 | (newlen << SDS_TYPE_BITS);
                }
                break;
            case SDS_TYPE_8:
                SDS_HDR(8,s)->len += inc;
                break;
            case SDS_TYPE_16:
                SDS_HDR(16,s)->len += inc;
                break;
            case SDS_TYPE_32:
                SDS_HDR(32,s)->len += inc;
                break;
            case SDS_TYPE_64:
                SDS_HDR(64,s)->len += inc;
                break;
        }
    }

    /* sdsalloc() = sdsavail() + sdslen() */
    static inline size_t sdsalloc(const sds s) {
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5:
                return SDS_TYPE_5_LEN(flags);
            case SDS_TYPE_8:
                return SDS_HDR(8,s)->alloc;
            case SDS_TYPE_16:
                return SDS_HDR(16,s)->alloc;
            case SDS_TYPE_32:
                return SDS_HDR(32,s)->alloc;
            case SDS_TYPE_64:
                return SDS_HDR(64,s)->alloc;
        }
        return 0;
    }

    static inline void sdssetalloc(sds s, size_t newlen) {
        unsigned char flags = s[-1];
        switch(flags&SDS_TYPE_MASK) {
            case SDS_TYPE_5:
                /* Nothing to do, this type has no total allocation info. */
                break;
            case SDS_TYPE_8:
                SDS_HDR(8,s)->alloc = newlen;
                break;
            case SDS_TYPE_16:
                SDS_HDR(16,s)->alloc = newlen;
                break;
            case SDS_TYPE_32:
                SDS_HDR(32,s)->alloc = newlen;
                break;
            case SDS_TYPE_64:
                SDS_HDR(64,s)->alloc = newlen;
                break;
        }
    }

### sds.c
创建一个sds:

    sds sdsnewlen(const void *init, size_t initlen) {
        void *sh;
        sds s;
        char type = sdsReqType(initlen);
        /* Empty strings are usually created in order to append. Use type 8
        * since type 5 is not good at this. */
        //这里是对创建空字符串的特殊处理，如果不是空字符串，则仍可能使用SDS_TYPE_5。
        if (type == SDS_TYPE_5 && initlen == 0) type = SDS_TYPE_8;
        int hdrlen = sdsHdrSize(type);
        unsigned char *fp; /* flags pointer. */
        分配的空间大小为header大小+字符串大小+末尾的空字符。
        是否要初始化
        if (init==SDS_NOINIT)
            init = NULL;
        在init为NULL的时候初始化为0。
        else if (!init)
            memset(sh, 0, hdrlen+initlen+1);
        if (sh == NULL) return NULL;
        s为分配的指针+header的长度，也就是char buf[]的地址。
        s = (char*)sh+hdrlen;
        fp是flags字段的地址。
        fp = ((unsigned char*)s)-1;
        设置flags和len及alloc，在这里len和alloc都一样，而对于SDS_TYPE_5并没有alloc的概念，也就是说，不会重新调整大小，len和alloc永远都是一样的。
        switch(type) {
            case SDS_TYPE_5: {
                *fp = type | (initlen << SDS_TYPE_BITS);
                break;
            }
            case SDS_TYPE_8: {
                SDS_HDR_VAR(8,s);
                sh->len = initlen;
                sh->alloc = initlen;
                *fp = type;
                break;
            }
            case SDS_TYPE_16: {
                SDS_HDR_VAR(16,s);
                sh->len = initlen;
                sh->alloc = initlen;
                *fp = type;
                break;
            }
            case SDS_TYPE_32: {
                SDS_HDR_VAR(32,s);
                sh->len = initlen;
                sh->alloc = initlen;
                *fp = type;
                break;
            }
            case SDS_TYPE_64: {
                SDS_HDR_VAR(64,s);
                sh->len = initlen;
                sh->alloc = initlen;
                *fp = type;
                break;
            }
        }
        如果initlen不为0，init不为NULL，则将其内容复制到sds中。
        if (initlen && init)
            memcpy(s, init, initlen);
        最后给sds结尾增加一个0.
        s[initlen] = '\0';
        return s;
    }
其余的操作也都是围绕sds和sdshdr来操作。  
当增加sds的容量时(sdsMakeRoomFor(sds s, size_t addlen))，如果需要会增加多余的容量以减少分配的次数。算法如下：

    len = sdslen(s);
    sh = (char*)s-sdsHdrSize(oldtype);
    newlen = (len+addlen);
    if (newlen < SDS_MAX_PREALLOC)
        newlen *= 2;
    else
        newlen += SDS_MAX_PREALLOC;
    其中SDS_MAX_PREALLOC为1M。也就是说扩到足够的空间后，如果总容量小于1M，那么就将容量再增加一倍，否则就将总容量再增加1M。

注意，在第一次初始化的时候并不会多分配容量，而是会分配initLen长度的容量，这样可以减少浪费。

redis-cli获取二进制数据时会将其转成可打印模式，但是试了没有办法从中恢复。 应该使用redis-cli get xxx > data这种方式来获取二进制文件。

