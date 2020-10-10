## C语言宏，可以参考https://gcc.gnu.org/onlinedocs/cpp/
### 想查看一些和系统编译器有关的宏，可以使用如下命令：gcc -dM -E - < /dev/null

### 想查看一些和glibc有关的宏，如\_\_GNU_LIBRARY\_\_,\_\_GLIBC\_\_ and \_\_GLIBC_MINOR\_\_ ，可以查看features.h，如/usr/include/features.h

### linux内核版本在/usr/includelinux/version.h，

- #define LINUX_VERSION_CODE 266002
- #define KERNEL_VERSION(a,b,c) (((a) << 16) + ((b) << 8) + (c))

### #用来把参数转换成字符串  
### 宏的变长参数列表，定义时使用...表示。定义中的最后一个命名参数后面的参数列表包括逗号会来替换宏中的__VA_ARGS__。  
例如：#define eprintf(…) fprintf (stderr, __VA_ARGS__)，  
也可以使用一个命名的变长参数，如#define eprintf(args…) fprintf (stderr, args)，如果使用了这种形式，就不能使用__VA_ARGS__。
https://gcc.gnu.org/onlinedocs/cpp/Variadic-Macros.html

### va_list用来处理函数的变长参数列表
    #include <stdarg.h>
    1. 首先声明一个va_list变量来存储变长参数列表。   
    2. 使用va_start来初始化va_list。 va_start是一个接受两个参数的宏，第一个参数为va_list,第二个参数为...之前的参数。  
    3. 使用va_arg来获取具体参数。va_arg接受两个参数，一个是va_list,一个是变量类型。va_arg会获取va_list中的下一个参数，参数类型即为传入的类型，然后继续往下移动一个。注意，必须知道所有的参数的类型，这也是为什么printf需要一个格式字符串。
    4. 使用va_end来清理va_list.
    如下所示：
    #include <stdarg.h>
    #include <stdio.h>
    double average ( int num, ... )
    {
        va_list arguments;                     
        double sum = 0;
    
        va_start ( arguments, num );           
        for ( int x = 0; x < num; x++ )        
        {
            sum += va_arg ( arguments, double ); 
        }
        va_end ( arguments ); 
        return sum / num;                      
    }

### ##是将两个符号连接成一个
例如下列宏，是将sdshdr和8（T为8）合成sdshdr8，这里T代表的是header的类型，s是一个sds。 
#define SDS_HDR_VAR(T,s) struct sdshdr##T *sh = (void*)((s)-(sizeof(struct sdshdr##T)));

