##eval性能测试

测试使用的客户端为Jedis3.0.1。客户端程序在个人电脑上，redis server在局域网10.10.0.130上。

从测试结果看，从应用方来看使用eval方式和使用redis命令性能差不多。这个应该是因为主要的时间消耗是在网络层面而不是redis执行上面。  
从理论上来说，如果都是使用单个请求-单个响应这种模型，那么影响较小，因为脚本解析在整体耗时中的占比较小，应该不超过5%；如果是大量使用pipeline方式,影响会较大。  

|测试方法|并发 | 每个线程执行次数 | 客户端平均执行时间(ms)| redis平均执行时间(us)|
|:--: | :--: | :--: | :-: | :--:|
|redisSet | 1 | 100000 | 2.22 | 6.61(set) 5.46(expire) |
|redisSet | 10 | 100000 | 2.34 | 2.46(set) 2.13(expire) |
|redisSet | 100 | 100000 | 3.28 | 0.90(set) 0.68(expire) |
|luaEvalSet | 1 | 100000 | 1.09 | 13.35(总的eval) 1.71(set) 0.58(expire)|
|luaEvalSet | 10 | 100000 | 1.46 | 23.38(总的eval) 2.53(set) 1.40(expire) |
|luaEvalSet | 100 | 100000 | 2.23 | 7.62(总的eval) 0.78(set) 0.43(expire) |
|luaEvalShaSet | 1 | 100000 | 1.04 |  43.49(总的evalsha) 5.28(set) 2.02(expire)|
|luaEvalShaSet | 10 | 100000 | 1.41 | 26.78(总的evalsha) 5.55(set) 3.32(expire)|
|redisExists | 1 | 100000 | 0.95 | 2.68|
|redisExists | 10 | 100000 |0.98  | 0.57 |
|luaExists | 1 | 100000 | 0.94 | 23.74(eval) 2.12(exists) |
|luaExists|10|100000|1.20|9.41(总的eval) 0.42(exists)|
|redisGet | 1 | 100000 | 1.01 | 4.98|
|redisGet | 10 | 100000 | 1.10 | 0.72 |
|luaGet | 1 | 100000 | 1.07 | 29.02(总的eval) 4.70(get)|
|luaGet | 10 | 100000 | 1.19 | 10.02(总的eval) 1.44(get) |


测试用代码如下：

    private static JedisOperation jedisOperation = JedisClient.getJedis();
	public static void testLuaEvalSet() {
		long s = System.currentTimeMillis();
		String script = "local rs = redis.call('set',KEYS[1],ARGV[1]);redis.call('expire',KEYS[1],ARGV[2]);return rs;";
		Object aObject = null;
		for (int i = 0; i < 100000; i++) {
			List<String> keys = new ArrayList<>();
			keys.add("xugmtest"+i);
			List<String> argvs = new ArrayList<>();
			String value = "ddddddddddddddddddddddddddd"+i;
			argvs.add(value);
			argvs.add("3600");
			aObject =jedisOperation.eval(script, keys, argvs);
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(aObject);
	}
    public static void testLuaEvalShaSet() {
		long s = System.currentTimeMillis();
		String script = "local rs = redis.call('set',KEYS[1],ARGV[1]);redis.call('expire',KEYS[1],ARGV[2]);return rs;";
		Object aObject = null;
		String sha = "707c3575c2317a3417ebcf5f16912715b7fbd8ff";
		for (int i = 0; i < 100000; i++) {
			List<String> keys = new ArrayList<>();
			keys.add("xugmtest"+i);
			List<String> argvs = new ArrayList<>();
			String value = "ddddddddddddddddddddddddddd"+i;
			argvs.add(value);
			argvs.add("3600");
			aObject =jedisOperation.evalsha(sha, keys, argvs);
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(aObject);
	}
	
	public static void testRedisSet() {
		long s = System.currentTimeMillis();
		Object aObject = null;
		for (int i = 0; i < 100000; i++) {
			String key = "xugmtest"+i;
			String value = "ddddddddddddddddddddddddddd"+i;
			aObject =jedisOperation.set(key, value);
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(aObject);
	}
	
	
	public static void testRedisGet() {
		long s = System.currentTimeMillis();
		Object aObject = null;
		String key = null;
		int hits = 0;
		for (int i = 0; i < 100000; i++) {
			key = "xugmtest"+i;
			aObject =jedisOperation.get(key);
			if (aObject != null) {
				hits++;
			}
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(hits);
	}

	public static void testLuaGet() {
		String script = "return redis.call('get',KEYS[1])";
		long s = System.currentTimeMillis();
		Object aObject = null;
		String key = null;
		int hits = 0;
		for (int i = 0; i < 100000; i++) {
			key = "xugmtest"+i;
			aObject =jedisOperation.eval(script,1,key);
			if (aObject != null) {
				hits++;
			}
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(hits);
	}
	
	public static void testLuaExists() {
		long s = System.currentTimeMillis();
		String script = "return redis.call('exists',KEYS[1])";
		Object aObject = null;
		for (int i = 0; i < 100000; i++) {
			String key = "xugmtest"+i;
			aObject = jedisOperation.eval(script, 1,key);
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(aObject);
	}
	
	public static void testRedisExists() {
		long s = System.currentTimeMillis();
		String key = "xugmtest";
		Object aObject = null;
		for (int i = 0; i < 100000; i++) {
			aObject = jedisOperation.exists(key);
		}
		long e = System.currentTimeMillis();
		System.out.println(e -s);
		System.out.println(aObject);
	}
