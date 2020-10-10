# Rax, an ANSI C radix tree implementation 见https://github.com/antirez/rax。

主要特性:

* 高效使用内存:
    + 打包的节点表示.
    + 如果键被设置为NULL，则能够避免在节点内设置NULL指针(在头中有一个`isnull`比特位).
    + 缺少父节点引用。当需要时，会使用一个栈来代替。
* 快速查找:
    + 边作为字节数组直接存在父节点中，当试图找到匹配时，不需要访问没用的子节点。比较其它实现，这种方式能增加缓存命中。
    + 通过将边作为两个单独的数组存储，一个边字符数组和一个边指针，扫描正确的子节点对缓存行友好。
* 完整的实现:
    + 删除时如果需要会重新压缩节点。
    + 迭代器 (包含了一种当树被修改同时使用迭代器的方法).
    + 随机行走迭代.
    + 包括和抵挡OOM的能力: 如果malloc()返回NULL，API可以报告一个OOM错误，并且保证树永远在一个一致状态。

一个节点的布局如下.在示例中，一个节点代表了一个键（所以关联了一个数据指针），并且有3个子节点 `x`, `y`, `z`.
在图示中，每个空格代表一个字节（也就是说HDR是4字节，xyz是3字节，x-ptr是8字节）。  
注意这里dataptr其实代表的是当前节点的父节点的数据，而不是当前节点的。  
也就是说节点的值指针是所有父节点构成的key对应的值，而节点本身的data[]字段实际代表的是子节点或者所有后代节点(压缩节点)对应的字符，这是理解Rax的一个关键点。

    +----+---+--------+--------+--------+--------+
    |HDR |xyz| x-ptr  | y-ptr  | z-ptr  |dataptr |
    +----+---+--------+--------+--------+--------+

头`HDR`是一个有着以下字段的位域:

    uint32_t iskey:1;     /* Does this node contain a key? */
    uint32_t isnull:1;    /* Associated value is NULL (don't store it). */
    uint32_t iscompr:1;   /* Node is compressed. */
    uint32_t size:29;     /* Number of children, or compressed string len. */
对于不是键并且正好有一个子节点的节点链，我们不使用如下方式存储：

    A -> B -> C -> [some other node]

而是用一个压缩的节点：

    "ABC" -> [some other node]

压缩节点的布局如下,注意压缩节点有且只有一个子节点，该子节点对应的是压缩节点的data[]字段的最后一个字符:

    +----+---+--------+
    |HDR |ABC|chld-ptr|
    +----+---+--------+

# 基本API

基本API是一个简单的字典，可以增加或删除元素。唯一的主要区别是插入和删除API同样接受可选的参数以返回旧的值的引用。

## 创建一棵基数树并且增加一个键

创建一个新的基数树:

    rax *rt = raxNew();

插入一个新的键:

    int raxInsert(rax *rax, unsigned char *s, size_t len, void *data,void **old);

例如:

    raxInsert(rt,(unsigned char*)"mykey",5,some_void_value,NULL);

如果键成功插入，则返回1；如果树种已经有了该键，则返回0，这种情况下，值会被更新；如果内存溢出，则也是返回0，只是`errno`被设置为`ENOMEM`.

如果关联的值`data`是NULL,存储键的节点不会使用额外的内存去存储NULL值，所以对于只有键组成的字典，如果使用NULL作为关联值，那么会节省内存。

注意，key是无符号字节数组，并且你需要指定长度：Rax是二进制安全的，所以键可以是任何值。

还有一些不会覆盖已经存在的键值对的插入函数:

    int raxTryInsert(rax *rax, unsigned char *s, size_t len, void *data,
                     void **old);

该函数和raxInsert()是一样的，除了如果键存在会返回0，而不会修改键的值。旧值仍然可以通过'old'指针返回。

## 键查找

查找函数如下:

    void *raxFind(rax *rax, unsigned char *s, size_t len);

如果键不存在，该函数返回一个特殊的值`raxNotFound`，所以用法如下:

    void *data = raxFind(rax,mykey,mykey_len);
    if (data == raxNotFound) return;
    printf("Key value is %p\n", data);

raxFind() 是一个只读函数，所以不会OOM，永远不会失败.

## 删除键

删除键同样可以返回键原先关联的值:

    int raxRemove(rax *rax, unsigned char *s, size_t len, void **old);

如果键被删除了，该函数返回1，否则如果键不存在，则返回0。该函数不会因为OOM而失败，但是如果在键被删除的时候发生了OOM，那么树节点不会重新压缩：在这种情况下基数树会不那么有效的编码。

# 迭代器

Rax的键是按照字典序排序的。

Rax迭代器允许通过不同的操作符找到一个指定的元素，然后通过`raxNext()`和`raxPrev()`在键空间中跳转。

## 基本迭代器使用

迭代器通常都是声明为局部变量，然后使用`raxStart`函数初始化:

    raxIterator iter;
    raxStart(&iter, rt); // Note that 'rt' is the radix tree pointer.

`raxStart`永远不会失败也不会返回值。迭代器一旦初始化，就可以从一个指定位置开始迭代。
使用`raxSeek`来达到这一目的:

    int raxSeek(raxIterator *it, unsigned char *ele, size_t len, const char *op);

例如，可以通过大于等于键`"foo"`来找到第一个元素:

    raxSeek(&iter,">=",(unsigned char*)"foo",3);

raxSeek()成功的时候返回1，失败的时候返回0。可能的失败有:

1. 传递了一个无效的操作符。
2. 寻找迭代器的时候发生了OOM.

一旦找到了迭代器，就可以使用像下面示例这样使用`raxNext`和`raxPrev`来进行迭代:

    while(raxNext(&iter)) {
        printf("Key: %.*s\n", (int)iter.key_len, (char*)iter.key);
    }

`raxNext`返回从`raxSeek`找到的元素开始的元素，直到树种的最后一个元素。
当没有更多元素时，返回0，否则返回1.但是在发生OOM的时候，也会返回0.

# 释放迭代器

迭代器可以被使用多次，可以使用一遍又一遍的使用`raxSeek`来查找，而不必再调用`raxStart`。但是当迭代器不再使用时，要用如下方式回收:

    raxStop(&iter);

注意即使不调用`raxStop`, 通常情况下你也不会检测到内存泄漏,但是这只是Rax实现的一个副作用:多数情况下，它会尝试使用在栈上分配的数据结构。
但是对于很深的树或者很大的键，会使用堆内存，如果不调用 `raxStop`，那么会导致内存泄漏。

## 查找操作符

`raxSeek`可以根据操作符找到不同的元素。

操作符集合:

* `==` seek the element exactly equal to the given one.
* `>` seek the element immediately greater than the given one.
* `>=` seek the element equal, or immediately greater than the given one.
* `<` seek the element immediately smaller than the given one.
* `<=` seek the element equal, or immediately smaller than the given one.
* `^` seek the smallest element of the radix tree.
* `$` seek the greatest element of the radix tree.

## 迭代器结束条件

有时，我们想迭代特定的范围，比如从AAA到BBB。Rax库提供了`raxCompare`函数，这样你就不用一遍又一遍的编写同样的字符串比较方法:

    raxIterator iter;
    raxStart(&iter);
    raxSeek(&iter,">=",(unsigned char*)"AAA",3); // Seek the first element
    while(raxNext(&iter)) {
        if (raxCompare(&iter,">",(unsigned char*)"BBB",3)) break;
        printf("Current key: %.*s\n", (int)iter.key_len,(char*)iter.key);
    }
    raxStop(&iter);

`raxCompare`函数的原型如下:

    int raxCompare(raxIterator *iter, const char *op, unsigned char *key, size_t key_len);

支持的操作符有`>`, `>=`, `<`, `<=`, `==`.

## 检查迭代器EOF条件

有时候我们在调用raxNext()或者raxPrev()之前想知道迭代器是否处于EOF状态。
当调用raxNext()或者raxPrev()不再有返回值的时候，迭代器就处于EOF状态。
可能是因为raxSeek()没有找到元素或者通过raxPrev()和raxNext()调用遍历完了整个树
可以使用如下函数来判断是否到了EOF:

    int raxEOF(raxIterator *it);

## 遍历的时候修改基数树

为了高效，Rax迭代器会缓存我们当前在的节点，这样在下一个迭代步骤，可以从它上一次离开的地方继续开始。
但是，在缓存的节点指针不再有效的时候，迭代器有足够的状态可以重新查找。
这一问题发生于在迭代的过程中我们想修改基数树。一个常见的模式是，比如，删除所有符合某一个条件的元素。
注意，这里说的并不是并发修改，而是在迭代过程中修改。

幸运的是，有一个非常简单的方法实现这一目标，并且只在需要时，也就是树确实被修改的时候才需要付出效率代价。
解决方案包括一旦树被修改了，使用当前的键重新查找迭代器，如下所示:

    while(raxNext(&iter,...)) {
        if (raxRemove(rax,...)) {
            raxSeek(&iter,">",iter.key,iter.key_size);
        }
    }

在上面这种情况，我们使用`raxNext`来进行迭代，所以我们是按照字典顺序往前进行。
每次我们移除一个元素，就需要使用当前元素和`>`操作符: 这样我们就会移向下一个元素，并且产生一个代表修改后的树的新状态。

同样的思想可以用于不同的场景:

* 在迭代时，每次增加或删除元素，迭代器都需要使用`raxSeek`重新查找.
* 当前迭代器的键永远是有效的，可以通过`iter.key_size`和`iter.key`来访问，即使被删除了也无所谓.

## Re-seeking iterators after EOF

当我们达到基数树的开始或者终点，迭代就到达了EOF。EOF条件是永久的，即使从反方向迭代也不会产生任何结果。

从迭代器返回的最后一个元素再次开始，继续迭代的最简单方法，解决是查找它自己：

    raxSeek(&iter,iter.key,iter.key_len,"==");

所以，例如为了写一个命令从头到尾打印一棵基数树的所有元素，然后再从尾到头打印，重复利用同一个迭代器，可以使用如下方式：

    raxSeek(&iter,"^",NULL,0);
    while(raxNext(&iter,NULL,0,NULL))
        printf("%.*s\n", (int)iter.key_len, (char*)iter.key);

    raxSeek(&iter,"==",iter.key,iter.key_len);
    while(raxPrev(&iter,NULL,0,NULL))
        printf("%.*s\n", (int)iter.key_len, (char*)iter.key);

## 随机元素选择

如果我们要求如下条件，那么从一棵基数树中根据相同概率获取元素就是不可能的：

1. 基数树比期望的小 (例如使用了允许元素排序的信息进行了扩大).
2. 我们想让操作快速，最坏是对数的 (这样像蓄水池取样之类的算法就不能用，因为它是O(N)的).

然后，对于或多或少已经平衡了的树，一个足够长的随机行走可以产生可接受的结果，并且速度够快，而且最终会返回每一个可能的元素，即使不是使用正确的概率。

为了进行随机行走，只要使用如下函数随便查找一个迭代器:

    int raxRandomWalk(raxIterator *it, size_t steps);

如果steps参数设置为0，那么该函数会走1到以2为底的树中元素数量的对数步，通常已经足够产生够好的结果。否则，你可以指定步数的精确数值。

## 打印树

可以使用如下函数打印树的ASCII艺术图形:

    raxShow(mytree);

但是注意，对元素数量比较少的树还可以，对很大的树会比较难以阅读。

下面是使用如下键值对的树，调用raxShow()的结果:

* alligator = (nil)
* alien = 0x1
* baloon = 0x2
* chromodynamic = 0x3
* romane = 0x4
* romanus = 0x5
* romulus = 0x6
* rubens = 0x7
* ruber = 0x8
* rubicon = 0x9
* rubicundus = 0xa
* all = 0xb
* rub = 0xc
* ba = 0xd

```
[abcr]
 `-(a) [l] -> [il]
               `-(i) "en" -> []=0x1
               `-(l) "igator"=0xb -> []=(nil)
 `-(b) [a] -> "loon"=0xd -> []=0x2
 `-(c) "hromodynamic" -> []=0x3
 `-(r) [ou]
        `-(o) [m] -> [au]
                      `-(a) [n] -> [eu]
                                    `-(e) []=0x4
                                    `-(u) [s] -> []=0x5
                      `-(u) "lus" -> []=0x6
        `-(u) [b] -> [ei]=0xc
                      `-(e) [nr]
                             `-(n) [s] -> []=0x7
                             `-(r) []=0x8
                      `-(i) [c] -> [ou]
                                    `-(o) [n] -> []=0x9
                                    `-(u) "ndus" -> []=0xa
```