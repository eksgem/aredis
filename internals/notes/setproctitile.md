## 设置进程名，将setproctitle.c

之所以要进行这一系列操作来修改进程名，是因为默认进程名就是argv[0]一直到argv[argc-1]中间使用空格分隔,但是argv和env在linux中是连续存储的，实际的存储结构为
argv[0]|argv[1]|argv[argc]为null|environ[0]|enrion[1]|最后一个为NULL|,所以不能直接修改argv[0]. 里面考虑了各种操作系统的情况，我们只考虑使用linux和__GLIBC__的情况，也就是需要使用下面的方法来实现修改进程名功能。所以需要将argv和env重新设置，并且将其原先所在位置设置为新的proctitle。

extern char **environ;

static struct {
	/* original value */
	const char *arg0;

	/* title space available */
	char *base, *end;

	 /* pointer to original nul character within base */
	char *nul;

	_Bool reset;
	int error;
} SPT;


#ifndef SPT_MIN
#define SPT_MIN(a, b) (((a) < (b))? (a) : (b))
#endif

static inline size_t spt_min(size_t a, size_t b) {
	return SPT_MIN(a, b);
} /* spt_min() */


static int spt_clearenv(void) {
	clearenv();
	return 0;
} /* spt_clearenv() */


static int spt_copyenv(char *oldenv[]) {
	extern char **environ;
	char *eq;
	int i, error;

	if (environ != oldenv)
		return 0;

	if ((error = spt_clearenv()))
		goto error;

	for (i = 0; oldenv[i]; i++) {
		if (!(eq = strchr(oldenv[i], '=')))
			continue;

		*eq = '\0';
        //使用setenv，是复制字符串，所以后面可以不用管之前系统初始化的env。
		error = (0 != setenv(oldenv[i], eq + 1, 1))? errno : 0;
		*eq = '=';

		if (error)
			goto error;
	}

	return 0;
error:
	environ = oldenv;

	return error;
} /* spt_copyenv() */


static int spt_copyargs(int argc, char *argv[]) {
	char *tmp;
	int i;

	for (i = 1; i < argc || (i >= argc && argv[i]); i++) {
		if (!argv[i])
			continue;
        // 使用strdup来复制字符串，这样后面就可以来修改argv[0]了。
		if (!(tmp = strdup(argv[i])))
			return errno;

		argv[i] = tmp;
	}

	return 0;
} /* spt_copyargs() */


void spt_init(int argc, char *argv[]) {
        char **envp = environ;
	char *base, *end, *nul, *tmp;
	int i, error;
    //base为原始的argv[0]的地址。
	if (!(base = argv[0]))
		return;
    // nul为argv[0]的null的地址。
	nul = &base[strlen(base)];
    // 此时end为argv[1]的地址.
	end = nul + 1;
    //这里的终止条件之一为argv[i]不为null，
    //所以当走到argv[argc]时，此循环会终止，这时候的end就是argv[argc]的地址。
    //这里使用这样复杂的判断，不知道是不是有些环境argc和argv并不匹配。
	for (i = 0; i < argc || (i >= argc && argv[i]); i++) {
		if (!argv[i] || argv[i] < end)
			continue;

		end = argv[i] + strlen(argv[i]) + 1;
	}
    //处理envp，envp没有长度字段，最后一个元素为NULL，在这里用做判断条件。
    //循环结束后，end的位置为envp的最后一个元素的地址。
	for (i = 0; envp[i]; i++) {
		if (envp[i] < end)
			continue;

		end = envp[i] + strlen(envp[i]) + 1;
	}

	if (!(SPT.arg0 = strdup(argv[0])))
		goto syerr;

	if (!(tmp = strdup(program_invocation_name)))
		goto syerr;

	program_invocation_name = tmp;

	if (!(tmp = strdup(program_invocation_short_name)))
		goto syerr;

	program_invocation_short_name = tmp;

	if ((error = spt_copyenv(envp)))
		goto error;
    // 在这之后argv的内容发生了变化
	if ((error = spt_copyargs(argc, argv)))
		goto error;

	SPT.nul  = nul;
	SPT.base = base;
	SPT.end  = end;

	return;
syerr:
	error = errno;
error:
	SPT.error = error;
} /* spt_init() */


#ifndef SPT_MAXTITLE
#define SPT_MAXTITLE 255
#endif

void setproctitle(const char *fmt, ...) {
	char buf[SPT_MAXTITLE + 1]; /* use buffer in case argv[0] is passed */
	va_list ap;
	char *nul;
	int len, error;
    //没有设置SPT表明没有初始化。
	if (!SPT.base)
		return;

	if (fmt) {
		va_start(ap, fmt);
		len = vsnprintf(buf, sizeof buf, fmt, ap);
		va_end(ap);
	} else {
		len = snprintf(buf, sizeof buf, "%s", SPT.arg0);
	}

	if (len <= 0)
		{ error = errno; goto error; }

	if (!SPT.reset) {
        // 将初始的argv和environ置为0.
		memset(SPT.base, 0, SPT.end - SPT.base);
		SPT.reset = 1;
	} else {
		memset(SPT.base, 0, spt_min(sizeof buf, SPT.end - SPT.base));
	}

	len = spt_min(len, spt_min(sizeof buf, SPT.end - SPT.base) - 1);
    //因为proctitle需要使用原来的位置，还是将proctitle设置回了原先的位置。前面的len判断已经确保了不会超过原先的argv+envp的大小。
	memcpy(SPT.base, buf, len);
	nul = &SPT.base[len];
    
	if (nul < SPT.nul) {
		*SPT.nul = '.';
	} else if (nul == SPT.nul && &nul[1] < SPT.end) {
		*SPT.nul = ' ';
		*++nul = '\0';
	}

	return;
error:
	SPT.error = error;
} /* setproctitle() */


