## 发布订阅

### 请参考，
https://redis.io/topics/pubsub  
https://redis.io/topics/notifications

### 发布类型，主要分为两种，
- 普通发布，也就是publish消息到某一个channel。在集群模式下该事件会广播到整个集群中；在非集群模式下,会传递到从中。所以在集群模式下，订阅一个节点就可以。
- 键空间/键事件发布，也就是数据变化时由redis发布的事件。该类型事件只会在本服务发布。如果想接收到全部键空间事件，需要在所有的节点上订阅。一个数据变化，实际会发送两个事件，方便根据情况订阅。
    - \_\_keyspace@\<db>\_\_:\<key> \<event>
    - \_\_keyevent@\<db>\_\_:\<event> \<key>

### 订阅类型，主要分为两种
- subscribe 订阅某些channel
- psubscribe 订阅某些模式的channel，模式是使用通配符（?,*）匹配。

### 注意事项

1. 消息发布是不可靠的，也就是说如果没有订阅者，则直接就丢弃了。另外，如果redis服务重启连订阅信息也会丢弃。  
所以，如果是长时间订阅，订阅后应该阻塞读取响应。如果连接进行了一些超时设置或者连接池有一些超时设置，那么连接有可能会被关闭；或者redis服务本身也可能会重启。在这种情况下，应该在一个死循环中创建连接（可能每次重新连接前等待一小段时间，防止死循环），并在这个连接上进行订阅。
2. 在客户端SUBSCRIBE后，该客户端就只能执行以下命令，SUBSCRIBE, PSUBSCRIBE, UNSUBSCRIBE, PUNSUBSCRIBE, PING and QUIT.  
所以，如果只是临时订阅一下并且使用了连接池，那么应该在还池前，调用一下UNSUBSCRIBE。  

### Jedis中发布订阅的实现

1. 单机版redis
构造一个JedisPubSub对象，里面是一些事件回调函数。然后通过Jedis.subscribe()方法来进行订阅。
订阅后，实际是在一个死循环中阻塞读取响应。但是注意这个死循环只是循环读取响应，并不会重新连接redis。  
2. 集群版redis
构造一个JedisPubSub对象，里面是一些事件回调函数。然后通过JedisCluster.subscribe()方法来进行订阅。实际是随机选择了一个节点进行了subscribe。根据上面的说明，对于普通事件是没问题的，但是无法处理键空间/键事件发布。

另外，注意Jedis接口只有subscribe和psubscribe，unsubscribe和punsubscribe都要通过JedisPubSub进行。对于集群来说，这种方式是必要的，因为集群每次是随机选择节点，而JedisPubSub实际是绑定了一个节点，所以可以精准的取消订阅。


