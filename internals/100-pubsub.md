## 发布订阅

### 请参考，
https://redis.io/topics/pubsub  
https://redis.io/topics/notifications

### 发布类型，主要分为两种，
- 普通发布，也就是publish消息到某一个channel。在集群模式下该事件会广播到整个集群中；在非集群模式下,会传递到从中。所以在集群模式下，订阅一个节点就可以。
- 键空间/键事件发布，也就是数据变化时由redis发布的事件。该类型事件只会在本服务发布。如果想接收到全部键空间事件，需要在所有的节点上订阅。一个数据变化，实际会发送两个事件，方便根据情况订阅。 
    - \_\_keyspace@\<db>\_\_:\<key> \<event>
    - \_\_keyevent@\<db>\_\_:\<event> \<key>  
另外，键空间事件默认并不开启，需要通过配置项notify-keyspace-events（配置文件或者CONFIG SET）开启。该配置项的值为一个字符串，有不同的字符分别代表不同的事件，可以进行组合，空字符串代表禁用，请参考上面参考文档。

### 订阅类型，主要分为两种
- subscribe 订阅某些channel
- psubscribe 订阅某些模式的channel，模式是使用通配符（?,*）匹配。

### redis记录subscribe/psubscribe的数据结构

publish信息不需要记录，可以随意发布。  
subscribe主要涉及到以下结构：
- server.pubsub_channels，订阅的channel。结构为一个字典，key为channel，值为一个clients的列表。发布的时候，直接通过channel查找到订阅的客户端列表，然后挨个发送通知。
- pubsub_patterns_dict，订阅的模式。结构为一个字段，key为订阅的模式，值为一个clients的列表。
发布的时候，需要遍历字典，如果模式匹配，则挨个给相应的客户端发送通知。从这里看，需要遍历所有的模式，所以模式如果过多的话，效率会较差。
- client上的pubsub_channels，结构为一个字段，key为channel，值为null。client上的pubsub_patterns，结果为一个列表，元素为pattern。  
client上的这两个字段，一是可以防止重复订阅，二是可以用来判断client是否处于订阅的上下文中。

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

另外，注意Jedis接口只有subscribe和psubscribe，unsubscribe和punsubscribe都要通过JedisPubSub进行。对于集群来说，这种方式是必要的，因为集群每次是随机选择节点，而JedisPubSub实际是绑定了一个节点，所以可以精准的取消订阅。另外，除了第一次的subscribe，后续的subscribe也必须通过JedisPubSub进行。

