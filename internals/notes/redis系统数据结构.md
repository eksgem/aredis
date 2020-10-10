## server.h

### struct redisServer记录了各种参数和运行状态。主要包括：
- pid，配置文件路径，执行文件路径，db，命令等基本信息
- 监听端口，连接的客户端等网络信息
- RDB，AOF相关存储信息
- 处理的命令数量，连接数量，接收和发送的字节数等统计信息
- slowlog
- redis.conf配置项
- 集群及主从相关信息
- 一些服务限制，比如最大客户端数，使用的内存数量等。
- pubsub相关信息
- lua脚本相关
- 其他一些

### struct client 记录了客户端的各种信息。
- 客户端id，socket，当前使用的db，存储请求的buffer，当前的命令及其参数，要返回给客户端的响应列表等
- 主从复制相关信息

### redisObject结构体
可以用来表示任意的redis数据结构，像strings, lists, sets, sorted sets等,
相比于原始数据结构，增加了类型，编码(从下面看目前在用的是有10种)，lru，引用计数等和管理相关的功能，这样原始数据结构只需专注于数据本身即可：

    typedef struct redisObject {
        unsigned type:4;
        unsigned encoding:4;
        unsigned lru:LRU_BITS; /* LRU time (relative to global lru_clock) or
                                * LFU data (least significant 8 bits frequency
                                * and most significant 16 bits access time). */
        int refcount;
        void *ptr;
    } robj;

    /* Get a decoded version of an encoded object (returned as a new object).
    * If the object is already raw-encoded just increment the ref count. */
    该函数实际就是将整数编码的对象转成字符串编码的。
    robj *getDecodedObject(robj *o) {
        从这里看decodedObject只可能是String或Int。
        robj *dec;

        if (sdsEncodedObject(o)) {
            incrRefCount(o);
            return o;
        }
        if (o->type == OBJ_STRING && o->encoding == OBJ_ENCODING_INT) {
            char buf[32];

            ll2string(buf,32,(long)o->ptr);
            dec = createStringObject(buf,strlen(buf));
            return dec;
        } else {
            serverPanic("Unknown encoding type");
        }
    }
    #define sdsEncodedObject(objptr) (objptr->encoding == OBJ_ENCODING_RAW || objptr->encoding == OBJ_ENCODING_EMBSTR)
    /* Objects encoding. Some kind of objects like Strings and Hashes can be
    * internally represented in multiple ways. The 'encoding' field of the object
    * is set to one of this fields for this object. */
    #define OBJ_ENCODING_RAW 0     /* Raw representation */
    #define OBJ_ENCODING_INT 1     /* Encoded as integer */
    #define OBJ_ENCODING_HT 2      /* Encoded as hash table */

    #define OBJ_ENCODING_ZIPMAP 3  /* Encoded as zipmap */
    #define OBJ_ENCODING_LINKEDLIST 4 /* No longer used: old list encoding. */
    #define OBJ_ENCODING_ZIPLIST 5 /* Encoded as ziplist */
    #define OBJ_ENCODING_INTSET 6  /* Encoded as intset */
    #define OBJ_ENCODING_SKIPLIST 7  /* Encoded as skiplist */
    #define OBJ_ENCODING_EMBSTR 8  /* Embedded sds string encoding */
    #define OBJ_ENCODING_QUICKLIST 9 /* Encoded as linked list of ziplists */
    #define OBJ_ENCODING_STREAM 10 /* Encoded as a radix tree of listpacks */
    从这里看，应该有些对象是共享的，其refcount为OBJ_SHARED_REFCOUNT即INT_MAX。
    void incrRefCount(robj *o) {
        if (o->refcount != OBJ_SHARED_REFCOUNT) o->refcount++;
    }
    #define OBJ_SHARED_REFCOUNT INT_MAX
    /* Create a string object with EMBSTR encoding if it is smaller than
    * OBJ_ENCODING_EMBSTR_SIZE_LIMIT, otherwise the RAW encoding is
    * used.
    *
    * The current limit of 44 is chosen so that the biggest string object
    * we allocate as EMBSTR will still fit into the 64 byte arena of jemalloc. */
    #define OBJ_ENCODING_EMBSTR_SIZE_LIMIT 44
    当字符串长度小于44的时候，使用嵌入式字符串。
    robj *createStringObject(const char *ptr, size_t len) {
        if (len <= OBJ_ENCODING_EMBSTR_SIZE_LIMIT)
            return createEmbeddedStringObject(ptr,len);
        else
            return createRawStringObject(ptr,len);
    }


    /* Create a string object with encoding OBJ_ENCODING_RAW, that is a plain
    * string object where o->ptr points to a proper sds string. */
    robj *createRawStringObject(const char *ptr, size_t len) {
        return createObject(OBJ_STRING, sdsnewlen(ptr,len));
    }

    /* Create a string object with encoding OBJ_ENCODING_EMBSTR, that is
    * an object where the sds string is actually an unmodifiable string
    * allocated in the same chunk as the object itself. */
    压缩版字符串就是将robj和sds的内存一次分配，然后放到一起。
    robj *createEmbeddedStringObject(const char *ptr, size_t len) {
        注意C语言的这种用法，分配的内存实际大于robj的需要，也包括了sds。
        robj *o = zmalloc(sizeof(robj)+sizeof(struct sdshdr8)+len+1);
        o+1的位置为robj后面的第一个字节。
        struct sdshdr8 *sh = (void*)(o+1);

        o->type = OBJ_STRING;
        o->encoding = OBJ_ENCODING_EMBSTR;
        sh+1的位置为去掉了sdshdr8这个头后面的实际字符串的位置。实际对sds来说也是使用这个位置，要找到真正的sds的位置要从这个位置倒推。
        o->ptr = sh+1;
        o->refcount = 1;
        if (server.maxmemory_policy & MAXMEMORY_FLAG_LFU) {
            o->lru = (LFUGetTimeInMinutes()<<8) | LFU_INIT_VAL;
        } else {
            o->lru = LRU_CLOCK();
        }

        sh->len = len;
        sh->alloc = len;
        sh->flags = SDS_TYPE_8;
        这个地方不用担心会真的有字符串叫"SDS_NOINIT"，因为这里比较的是地址。
        if (ptr == SDS_NOINIT)
            sh->buf[len] = '\0';
        else if (ptr) {
            memcpy(sh->buf,ptr,len);
            sh->buf[len] = '\0';
        } else {
            如果为NULL，则返回一个buf全置为null的sds。
            memset(sh->buf,0,len+1);
        }
        return o;
    }

    robj *createObject(int type, void *ptr) {
        robj *o = zmalloc(sizeof(*o));
        o->type = type;
        o->encoding = OBJ_ENCODING_RAW;
        o->ptr = ptr;
        o->refcount = 1;
        根据evict策略不同，设置不同的数值。这应该意味着evict策略不能动态设置。
        /* Set the LRU to the current lruclock (minutes resolution), or
        * alternatively the LFU counter. */
        LRU是最近最少使用，LFU是一定时期内被访问次数最少。
        if (server.maxmemory_policy & MAXMEMORY_FLAG_LFU) {
            o->lru = (LFUGetTimeInMinutes()<<8) | LFU_INIT_VAL;
        } else {
            o->lru = LRU_CLOCK();
        }
        return o;
    }


