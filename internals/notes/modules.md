## 模块

### 如何找到模块相关API,以StringPtrLen为例。

在服务启动过程中注册模块相关API：

    void moduleRegisterCoreAPI(void) {
        REGISTER_API(StringPtrLen);// 会变成moduleRegisterApi("RedisModule_StringPtrLen", (void *)(unsigned long)RM_StringPtrLen);

    //不知道为什么要先转成unsigned long再转成void *。
    #define REGISTER_API(name) \
    moduleRegisterApi("RedisModule_" #name, (void *)(unsigned long)RM_ ## name)
    即moduleRegisterApi("RedisModule_StringPtrLen", (void *)(unsigned long)RM_StringPtrLen)

    而moduleRegisterApi最终是将API的名称作为key，函数指针作为值加到server.moduleapi这个字典中。因为每个函数的签名不一样，所以这里要使用void *。
    int moduleRegisterApi(const char *funcname, void *funcptr) {
        return dictAdd(server.moduleapi, (char*)funcname, funcptr);
    }

    RM_StringPtrLen是一个函数。
        const char *RM_StringPtrLen(const RedisModuleString *str, size_t *len) {
            if (str == NULL) {
                const char *errmsg = "(NULL string reply referenced in module)";
                if (len) *len = strlen(errmsg);
                return errmsg;
            }
            if (len) *len = sdslen(str->ptr);
            return str->ptr;
        }

模块调用RedisModule_StringPtrLen:
        size_t len;
        const char *key = RedisModule_StringPtrLen(argv[0], &len);
解析如下宏，
    #define REDISMODULE_API_FUNC(x) (*x)
    const char *REDISMODULE_API_FUNC(RedisModule_StringPtrLen)(const RedisModuleString *str, size_t *len);
    变成 const char * (*RedisModule_StringPtrLen)(const RedisModuleString *str, size_t *len); 
    这句的意思是声明一个名字为RedisModule_StringPtrLen，参数为const RedisModuleString * 和 size_t *,返回值为const char *的函数指针。
    也就是说RedisModule_StringPtrLen是在这里声明的，而其赋值则是在RedisModule_Init中。

    同样RedisModule_GetApi，也是由这样来声明得到：
    int REDISMODULE_API_FUNC(RedisModule_GetApi)(const char *, void *);
    变成int (*RedisModule_GetApi)(const char *, void *);
    也就是声明一个名字为RedisModule_GetApi，参数为const char * 和 void *，返回值为int的函数指针。


而RedisModule_GetApi是在模块初始化时候由RedisModuleCtx的第一个字段传入的。最终会通过该函数获取实际的函数指针。
    static int RedisModule_Init(RedisModuleCtx *ctx, const char *name, int ver, int apiver) {
    // 不知道为什么使用这种方式转换，而不是使用成员变量。
    void *getapifuncptr = ((void**)ctx)[0];
    // 将getapifuncptr转换为一个参数为const char * 和 void *，返回值为int的函数指针。
    RedisModule_GetApi = (int (*)(const char *, void *)) (unsigned long)getapifuncptr;

    REDISMODULE_GET_API(StringPtrLen);
    #define REDISMODULE_GET_API(name) \
        RedisModule_GetApi("RedisModule_" #name, ((void **)&RedisModule_ ## name))
    通过上面两步得到RedisModule_GetApi("RedisModule_StringPtrLen",((void**))&RedisModule_StringPtrLen);
    也就是通过RedisModule_GetApi获得"RedisModule_StringPtrLen"这个API对应的函数指针，并将其设置到RedisModule_StringPtrLen上。

    而RedisModule_GetApi这个函数由RedisModule_Init中的RedisModuleCtx *参数传入，RedisModule_Init又是由RedisModule_OnLoad调用的。
    RedisModule_OnLoad由模块载入系统调用，传入的RedisModuleCtx *为REDISMODULE_CTX_INIT。
    
    REDISMODULE_CTX_INIT的定义如下：
    #define REDISMODULE_CTX_INIT {(void*)(unsigned long)&RM_GetApi, NULL, NULL, NULL, NULL, 0, 0, 0, NULL, 0, NULL, NULL, 0, NULL}
    也就是说，最终是调用RM_GetApi来获取函数指针，而对于StringPtrLen其即为RM_StringPtrLen。

    而RM_GetApi就是从启动过程中注册的server.moduleapi字典中找到对应的函数指针，并将其设置到targetPtrPtr中。
    int RM_GetApi(const char *funcname, void **targetPtrPtr) {
        dictEntry *he = dictFind(server.moduleapi, funcname);
        if (!he) return REDISMODULE_ERR;
        *targetPtrPtr = dictGetVal(he);
        return REDISMODULE_OK;
    }
    
### OOM 

oom处理是在server.c的processCommand()中进行的。所以如果模块扩展命令要增加内存使用，应该设置deny-oom.
deny-oom标记是在RM_CreateCommand()的时候由commandFlagsFromString()加到command的flags中，(!strcasecmp(t,"deny-oom")) flags |= CMD_DENYOOM;。

### Cluster处理

对cluster的处理也是在server.c的processCommand()中进行的。所以内部不需要关心slot转移。
另外，如果命令不处理key，那么command的firstKey应该设置为0.

### redis模块可以注册拦截器

见RM_RegisterCommandFilter函数说明。