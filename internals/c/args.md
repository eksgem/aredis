## C语言的argc和argv

### main函数可以有两种形式：

- 没有任何参数 

int main(void) { /*...*/ }

- 有两个参数

int main(int argc, char *argv[]) { /*...*/ }
注意argv[argc]为NULL。通常argv[0]为执行程序的名字，但是也不是必须是，要看调用方如何调用execve系统调用。其他exec函数都不是系统调用，最终会调用execve。例如execl("/var/tmp/t", "test", NULL);执行的程序为t，但是argv[0]为test，在ps和top等程序中也会显示test。

        #include <unistd.h>
        int execve(const char *pathname, char *const argv[], char *const envp[]);

        argv是一个字符串数组，作为传给新程序的参数。通常argv[0]应该是执行的程序的文件名。envp是是一个字符串数组，作为传给新程序的环境变量，通常形式为key=value。argv和envp都要以一个NULL元素结尾。
 
实际在linux下，如下函数也是合法的：  
int main(int argc, char *argv[], char *envp[]){}  
不过根据POSIX.1标准，环境变量应该通过外部变量environ传入。environ的使用方式如下：
- 应用程序声明extern char **environ; 使用environ。
- #define _GNU_SOURCE  
  #include <unistd.h>
  使用environ。 注意_GNU_SOURCE应该在引入header之前定义。在unistd.h内部声明environ的时候实际是判断__USE_GNU这个宏。但是这个宏一般是内部用的，应用程序应该使用_GNU_SOURCE,unistd.h会包含features.h,在features.h中会根据_GNU_SOURCE来定义__USE_GNU。_GNU_SOURCE还会同时开启别的很多features。  
  见https://linux.die.net/man/7/feature_test_macros
- #include <unistd.h>,使用__environ。



