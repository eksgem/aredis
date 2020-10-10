## 字典相关数据结构
字典是redis中核心的数据结构。
字典的元素包含一个key，不同类型的值和指向下一个项的指针。  
typedef struct dictEntry {
    void *key;
    union {
        void *val;
        uint64_t u64;
        int64_t s64;
        double d;
    } v;
    struct dictEntry *next;
} dictEntry;

dictType定义了对字典进行操作的各种函数。hashFunction用来生成hash值，keyDup用来复制key，valDup用来复制value，keyCompare用来比较key，keyDustructor用来析构key,valDestructor用来析构value。privdata 属性则保存了需要传给那些类型特定函数的可选参数。
typedef struct dictType {
    uint64_t (*hashFunction)(const void *key);
    void *(*keyDup)(void *privdata, const void *key);
    void *(*valDup)(void *privdata, const void *obj);
    int (*keyCompare)(void *privdata, const void *key1, const void *key2);
    void (*keyDestructor)(void *privdata, void *key);
    void (*valDestructor)(void *privdata, void *obj);
} dictType;

一个hash table包括，dictEntry指针的数组，size为总的slot的数量，是2的幂,sizemask用来帮助获取index，为size-1，used为元素的数量。
/* This is our hash table structure. Every dictionary has two of this as we
 * implement incremental rehashing, for the old to the new table. */
typedef struct dictht {
    dictEntry **table;
    unsigned long size;
    unsigned long sizemask;
    unsigned long used;
} dictht;

最终真正的字典，包括dictType,两个hash table，rehashidx表明是否在重建hash。
typedef struct dict {
    dictType *type;
    void *privdata;
    dictht ht[2];
    long rehashidx; /* rehashing not in progress if rehashidx == -1 */
    unsigned long iterators; /* number of iterators currently running */
} dict;


/* If safe is set to 1 this is a safe iterator, that means, you can call
 * dictAdd, dictFind, and other functions against the dictionary even while
 * iterating. Otherwise it is a non safe iterator, and only dictNext()
 * should be called while iterating. */
typedef struct dictIterator {
    dict *d;
    long index;
    int table, safe;
    dictEntry *entry, *nextEntry;
    /* unsafe iterator fingerprint for misuse detection. */
    long long fingerprint;
} dictIterator;

/* This is the initial size of every hash table */
#define DICT_HT_INITIAL_SIZE     4

/* ------------------------------- Macros ------------------------------------*/
#define dictFreeVal(d, entry) \
    if ((d)->type->valDestructor) \
        (d)->type->valDestructor((d)->privdata, (entry)->v.val)

#define dictSetVal(d, entry, _val_) do { \
    if ((d)->type->valDup) \
        (entry)->v.val = (d)->type->valDup((d)->privdata, _val_); \
    else \
        (entry)->v.val = (_val_); \
} while(0)

#define dictSetSignedIntegerVal(entry, _val_) \
    do { (entry)->v.s64 = _val_; } while(0)

#define dictSetUnsignedIntegerVal(entry, _val_) \
    do { (entry)->v.u64 = _val_; } while(0)

#define dictSetDoubleVal(entry, _val_) \
    do { (entry)->v.d = _val_; } while(0)

#define dictFreeKey(d, entry) \
    if ((d)->type->keyDestructor) \
        (d)->type->keyDestructor((d)->privdata, (entry)->key)

#define dictSetKey(d, entry, _key_) do { \
    if ((d)->type->keyDup) \
        (entry)->key = (d)->type->keyDup((d)->privdata, _key_); \
    else \
        (entry)->key = (_key_); \
} while(0)

#define dictCompareKeys(d, key1, key2) \
    (((d)->type->keyCompare) ? \
        (d)->type->keyCompare((d)->privdata, key1, key2) : \
        (key1) == (key2))

#define dictHashKey(d, key) (d)->type->hashFunction(key)
#define dictGetKey(he) ((he)->key)
#define dictGetVal(he) ((he)->v.val)
#define dictGetSignedIntegerVal(he) ((he)->v.s64)
#define dictGetUnsignedIntegerVal(he) ((he)->v.u64)
#define dictGetDoubleVal(he) ((he)->v.d)
#define dictSlots(d) ((d)->ht[0].size+(d)->ht[1].size)
#define dictSize(d) ((d)->ht[0].used+(d)->ht[1].used)
#define dictIsRehashing(d) ((d)->rehashidx != -1)

 This file implements in memory hash tables with insert/del/replace/find/
 * get-random-element operations. Hash tables will auto resize if needed
 * tables of power of two in size are used, collisions are handled by
 * chaining.

 /* Using dictEnableResize() / dictDisableResize() we make possible to
 * enable/disable resizing of the hash table as needed. This is very important
 * for Redis, as we use copy-on-write and don't want to move too much memory
 * around when there is a child performing saving operations.
 *
 * Note that even when dict_can_resize is set to 0, not all resizes are
 * prevented: a hash table is still allowed to grow if the ratio between
 * the number of elements and the buckets > dict_force_resize_ratio. */
static int dict_can_resize = 1;
static unsigned int dict_force_resize_ratio = 5;

默认的hash函数是使用的SipHash，从中可以看到生成hash的过程中用到了一个种子，该种子是在每次服务启动的时候随机数生成器生成的。
从这点上来看，redis不能够使用facebook对memcache使用共享内存来实现重启服务而不丢失内存数据的这种方式。
另外，因为key的hash在每次服务运行时都不一样，所以RDB中也不是直接存储整个内存数据结构，而是循环存储key和value,同时处理expire等信息，当然也没办法直接存储整个数据结构，因为都是通过系统分配的内存，而不是redis自己从操作系统申请一整块内存，然后内部分配。
uint64_t dictGenHashFunction(const void *key, int len) {
    return siphash(key,len,dict_hash_function_seed);
}

rehash,就是分步骤将ht[0]中的数据转到ht[1]中。假设现在处于rehashing过程中，那么如果是查找，则是查找ht[0]和ht[1];如果是设置则是设置到ht[1]中。  
int dictRehash(dict *d, int n) {
    int empty_visits = n*10; /* Max number of empty buckets to visit. */
    if (!dictIsRehashing(d)) return 0;

    while(n-- && d->ht[0].used != 0) {
        dictEntry *de, *nextde;

        /* Note that rehashidx can't overflow as we are sure there are more
         * elements because ht[0].used != 0 */
        assert(d->ht[0].size > (unsigned long)d->rehashidx);
        while(d->ht[0].table[d->rehashidx] == NULL) {
            d->rehashidx++;
            if (--empty_visits == 0) return 1;
        }
        de = d->ht[0].table[d->rehashidx];
        /* Move all the keys in this bucket from the old to the new hash HT */
        while(de) {
            uint64_t h;

            nextde = de->next;
            /* Get the index in the new hash table */
            h = dictHashKey(d, de->key) & d->ht[1].sizemask;
            de->next = d->ht[1].table[h];
            d->ht[1].table[h] = de;
            d->ht[0].used--;
            d->ht[1].used++;
            de = nextde;
        }
        d->ht[0].table[d->rehashidx] = NULL;
        d->rehashidx++;
    }

    /* Check if we already rehashed the whole table... */
    if (d->ht[0].used == 0) {
        zfree(d->ht[0].table);
        d->ht[0] = d->ht[1];
        _dictReset(&d->ht[1]);
        d->rehashidx = -1;
        return 0;
    }

    /* More to rehash... */
    return 1;
}

增加一个元素，如果dict正在rehashing，那么执行一步rehash，这样是为了分散压力。
如果key不存在，则返回新建的dictEntry*，否则返回NULL。
/* Low level add or find:
 * This function adds the entry but instead of setting a value returns the
 * dictEntry structure to the user, that will make sure to fill the value
 * field as he wishes.
 *
 * This function is also directly exposed to the user API to be called
 * mainly in order to store non-pointers inside the hash value, example:
 *
 * entry = dictAddRaw(dict,mykey,NULL);
 * if (entry != NULL) dictSetSignedIntegerVal(entry,1000);
 *
 * Return values:
 *
 * If key already exists NULL is returned, and "*existing" is populated
 * with the existing entry if existing is not NULL.
 *
 * If key was added, the hash entry is returned to be manipulated by the caller.
 */
dictEntry *dictAddRaw(dict *d, void *key, dictEntry **existing)
{
    long index;
    dictEntry *entry;
    dictht *ht;

    if (dictIsRehashing(d)) _dictRehashStep(d);

    /* Get the index of the new element, or -1 if
     * the element already exists. */
    if ((index = _dictKeyIndex(d, key, dictHashKey(d,key), existing)) == -1)
        return NULL;

    /* Allocate the memory and store the new entry.
     * Insert the element in top, with the assumption that in a database
     * system it is more likely that recently added entries are accessed
     * more frequently. */
    ht = dictIsRehashing(d) ? &d->ht[1] : &d->ht[0];
    entry = zmalloc(sizeof(*entry));
    entry->next = ht->table[index];
    ht->table[index] = entry;
    ht->used++;

    /* Set the hash entry fields. */
    dictSetKey(d, entry, key);
    return entry;
}

字典扩充的过程主要是将ht[1]扩展到当前used的下一个2的幂次。
rehash的过程基本上是设置rehashing标记，在rehashing过程中，如果添加元素，则添加到ht[1]中，如果查找元素则在两个表中都查找。
在rehash完成后就可以将ht[0]释放，将ht[1]设置到ht1[0]上。
/* Expand or create the hash table */
int dictExpand(dict *d, unsigned long size)
{
    /* the size is invalid if it is smaller than the number of
     * elements already inside the hash table */
    if (dictIsRehashing(d) || d->ht[0].used > size)
        return DICT_ERR;

    dictht n; /* the new hash table */
    unsigned long realsize = _dictNextPower(size);

    /* Rehashing to the same table size is not useful. */
    if (realsize == d->ht[0].size) return DICT_ERR;

    /* Allocate the new hash table and initialize all pointers to NULL */
    n.size = realsize;
    n.sizemask = realsize-1;
    n.table = zcalloc(realsize*sizeof(dictEntry*));
    n.used = 0;

    /* Is this the first initialization? If so it's not really a rehashing
     * we just set the first hash table so that it can accept keys. */
    if (d->ht[0].table == NULL) {
        d->ht[0] = n;
        return DICT_OK;
    }

    /* Prepare a second hash table for incremental rehashing */
    d->ht[1] = n;
    d->rehashidx = 0;
    return DICT_OK;
}

existing是用来做结果返回的。
/* Returns the index of a free slot that can be populated with
 * a hash entry for the given 'key'.
 * If the key already exists, -1 is returned
 * and the optional output parameter may be filled.
 *
 * Note that if we are in the process of rehashing the hash table, the
 * index is always returned in the context of the second (new) hash table. */
static long _dictKeyIndex(dict *d, const void *key, uint64_t hash, dictEntry **existing)
{
    unsigned long idx, table;
    dictEntry *he;
    if (existing) *existing = NULL;

    /* Expand the hash table if needed */
    if (_dictExpandIfNeeded(d) == DICT_ERR)
        return -1;
    for (table = 0; table <= 1; table++) {
        idx = hash & d->ht[table].sizemask;
        /* Search if this slot does not already contain the given key */
        he = d->ht[table].table[idx];
        while(he) {
            if (key==he->key || dictCompareKeys(d, key, he->key)) {
                if (existing) *existing = he;
                return -1;
            }
            he = he->next;
        }
        if (!dictIsRehashing(d)) break;
    }
    return idx;
}

通过迭代器获取下一个元素。 

    dictEntry *dictNext(dictIterator *iter)
    {
        while (1) {
            if (iter->entry == NULL) {
                dictht *ht = &iter->d->ht[iter->table];
                初始状态
                if (iter->index == -1 && iter->table == 0) {
                    如果是safe则将dict的并发迭代器数加1，否则通过本迭代器记录dict的fingerprint
                    if (iter->safe)
                        iter->d->iterators++;
                    else
                        iter->fingerprint = dictFingerprint(iter->d);
                }
                iter->index++;
                if (iter->index >= (long) ht->size) {
                    找完了ht[0]开始找ht[1]
                    if (dictIsRehashing(iter->d) && iter->table == 0) {
                        iter->table++;
                        iter->index = 0;
                        ht = &iter->d->ht[1];
                    } else {
                        break;
                    }
                }
                iter->entry = ht->table[iter->index];
            } else {
                iter->entry = iter->nextEntry;
            }
            if (iter->entry) {
                /* We need to save the 'next' here, the iterator user
                * may delete the entry we are returning. */
                iter->nextEntry = iter->entry->next;
                return iter->entry;
            }
        }
        return NULL;
    }


从这里可以看到dict不能太稀疏，不然取random key可能会比较耗时。     

    /* Return a random entry from the hash table. Useful to
    * implement randomized algorithms */
    dictEntry *dictGetRandomKey(dict *d)
    {
        dictEntry *he, *orighe;
        unsigned long h;
        int listlen, listele;

        if (dictSize(d) == 0) return NULL;
        if (dictIsRehashing(d)) _dictRehashStep(d);
        if (dictIsRehashing(d)) {
            do {
                /* We are sure there are no elements in indexes from 0
                * to rehashidx-1 */
                h = d->rehashidx + (random() % (d->ht[0].size +
                                                d->ht[1].size -
                                                d->rehashidx));
                he = (h >= d->ht[0].size) ? d->ht[1].table[h - d->ht[0].size] :
                                        d->ht[0].table[h];
            } while(he == NULL);
        } else {
            do {
                h = random() & d->ht[0].sizemask;
                he = d->ht[0].table[h];
            } while(he == NULL);
        }

        /* Now we found a non empty bucket, but it is a linked
        * list and we need to get a random element from the list.
        * The only sane way to do so is counting the elements and
        * select a random index. */
        listlen = 0;
        orighe = he;
        while(he) {
            he = he->next;
            listlen++;
        }
        listele = random() % listlen;
        he = orighe;
        while(listele--) he = he->next;
        return he;
    }

redis在很多能用int的地方使用了unsigned long，不知道是不是为了性能考虑。

比特反转，也就是将高位和低位的bit进行交换。
/* Function to reverse bits. Algorithm from:
 * http://graphics.stanford.edu/~seander/bithacks.html#ReverseParallel */
static unsigned long rev(unsigned long v) {
    unsigned long s = 8 * sizeof(v); // bit size; must be power of 2
    unsigned long mask = ~0;
    while ((s >>= 1) > 0) {
        mask ^= (mask << s);
        v = ((v >> s) & mask) | ((v << s) & ~mask);
    }
    return v;
}

https://blog.csdn.net/gqtcgq/article/details/50533336  
https://github.com/antirez/redis/pull/579#issuecomment-16871583

    /* dictScan() is used to iterate over the elements of a dictionary.
    *
    * Iterating works the following way:
    *
    * 1) Initially you call the function using a cursor (v) value of 0.
    * 2) The function performs one step of the iteration, and returns the
    *    new cursor value you must use in the next call.
    * 3) When the returned cursor is 0, the iteration is complete.
    *
    * The function guarantees all elements present in the
    * dictionary get returned between the start and end of the iteration.
    * However it is possible some elements get returned multiple times.
    *
    * For every element returned, the callback argument 'fn' is
    * called with 'privdata' as first argument and the dictionary entry
    * 'de' as second argument.
    *
    * HOW IT WORKS.
    *
    * The iteration algorithm was designed by Pieter Noordhuis.
    * The main idea is to increment a cursor starting from the higher order
    * bits. That is, instead of incrementing the cursor normally, the bits
    * of the cursor are reversed, then the cursor is incremented, and finally
    * the bits are reversed again.
    *
    * This strategy is needed because the hash table may be resized between
    * iteration calls.
    *
    * dict.c hash tables are always power of two in size, and they
    * use chaining, so the position of an element in a given table is given
    * by computing the bitwise AND between Hash(key) and SIZE-1
    * (where SIZE-1 is always the mask that is equivalent to taking the rest
    *  of the division between the Hash of the key and SIZE).
    *
    * For example if the current hash table size is 16, the mask is
    * (in binary) 1111. The position of a key in the hash table will always be
    * the last four bits of the hash output, and so forth.
    *
    * WHAT HAPPENS IF THE TABLE CHANGES IN SIZE?
    *
    * If the hash table grows, elements can go anywhere in one multiple of
    * the old bucket: for example let's say we already iterated with
    * a 4 bit cursor 1100 (the mask is 1111 because hash table size = 16).
    *
    * If the hash table will be resized to 64 elements, then the new mask will
    * be 111111. The new buckets you obtain by substituting in ??1100
    * with either 0 or 1 can be targeted only by keys we already visited
    * when scanning the bucket 1100 in the smaller hash table.
    *
    * By iterating the higher bits first, because of the inverted counter, the
    * cursor does not need to restart if the table size gets bigger. It will
    * continue iterating using cursors without '1100' at the end, and also
    * without any other combination of the final 4 bits already explored.
    *
    * Similarly when the table size shrinks over time, for example going from
    * 16 to 8, if a combination of the lower three bits (the mask for size 8
    * is 111) were already completely explored, it would not be visited again
    * because we are sure we tried, for example, both 0111 and 1111 (all the
    * variations of the higher bit) so we don't need to test it again.
    *
    * WAIT... YOU HAVE *TWO* TABLES DURING REHASHING!
    *
    * Yes, this is true, but we always iterate the smaller table first, then
    * we test all the expansions of the current cursor into the larger
    * table. For example if the current cursor is 101 and we also have a
    * larger table of size 16, we also test (0)101 and (1)101 inside the larger
    * table. This reduces the problem back to having only one table, where
    * the larger one, if it exists, is just an expansion of the smaller one.
    *
    * LIMITATIONS
    *
    * This iterator is completely stateless, and this is a huge advantage,
    * including no additional memory used.
    *
    * The disadvantages resulting from this design are:
    *
    * 1) It is possible we return elements more than once. However this is usually
    *    easy to deal with in the application level.
    * 2) The iterator must return multiple elements per call, as it needs to always
    *    return all the keys chained in a given bucket, and all the expansions, so
    *    we are sure we don't miss keys moving during rehashing.
    * 3) The reverse cursor is somewhat hard to understand at first, but this
    *    comment is supposed to help.
    */
    unsigned long dictScan(dict *d,
                        unsigned long v,
                        dictScanFunction *fn,
                        dictScanBucketFunction* bucketfn,
                        void *privdata)
    {
        dictht *t0, *t1;
        const dictEntry *de, *next;
        unsigned long m0, m1;

        if (dictSize(d) == 0) return 0;

        if (!dictIsRehashing(d)) {
            t0 = &(d->ht[0]);
            m0 = t0->sizemask;

            /* Emit entries at cursor */
            if (bucketfn) bucketfn(privdata, &t0->table[v & m0]);
            de = t0->table[v & m0];
            while (de) {
                next = de->next;
                fn(privdata, de);
                de = next;
            }

            /* Set unmasked bits so incrementing the reversed cursor
            * operates on the masked bits */
            v |= ~m0;

            /* Increment the reverse cursor */
            v = rev(v);
            v++;
            v = rev(v);

        } else {
            t0 = &d->ht[0];
            t1 = &d->ht[1];

            /* Make sure t0 is the smaller and t1 is the bigger table */
            if (t0->size > t1->size) {
                t0 = &d->ht[1];
                t1 = &d->ht[0];
            }

            m0 = t0->sizemask;
            m1 = t1->sizemask;

            /* Emit entries at cursor */
            if (bucketfn) bucketfn(privdata, &t0->table[v & m0]);
            de = t0->table[v & m0];
            while (de) {
                next = de->next;
                fn(privdata, de);
                de = next;
            }

            /* Iterate over indices in larger table that are the expansion
            * of the index pointed to by the cursor in the smaller table */
            do {
                /* Emit entries at cursor */
                if (bucketfn) bucketfn(privdata, &t1->table[v & m1]);
                de = t1->table[v & m1];
                while (de) {
                    next = de->next;
                    fn(privdata, de);
                    de = next;
                }

                /* Increment the reverse cursor not covered by the smaller mask.*/
                v |= ~m1;
                v = rev(v);
                v++;
                v = rev(v);

                /* Continue while bits covered by mask difference is non-zero */
                第一次循环的时候，小于m0 ^ m1(即小于等于m0)的v已经处理过了。后面处理的都是大于m0的v。根据rehash的算法，小的table扩展到大的table，index之间的对应关系是已知的。 比如size为8的table(mask 111)，对应size为16的table(mask 1111)的index即为高位为1或0，低位为index。 而根据反转再加1再反转的算法，最开始的时候，高位为全0，循环一遍后再次回到全0，那么循环结束。 
            } while (v & (m0 ^ m1));
        }

        return v;
    }

扩展dict。
    /* Expand the hash table if needed */
    static int _dictExpandIfNeeded(dict *d)
    {
        /* Incremental rehashing already in progress. Return. */
        if (dictIsRehashing(d)) return DICT_OK;

        /* If the hash table is empty expand it to the initial size. */
        if (d->ht[0].size == 0) return dictExpand(d, DICT_HT_INITIAL_SIZE);

        /* If we reached the 1:1 ratio, and we are allowed to resize the hash
        * table (global setting) or we should avoid it but the ratio between
        * elements/buckets is over the "safe" threshold, we resize doubling
        * the number of buckets. */
        if (d->ht[0].used >= d->ht[0].size &&
            (dict_can_resize ||
            d->ht[0].used/d->ht[0].size > dict_force_resize_ratio))
        {
            return dictExpand(d, d->ht[0].used*2);
        }
        return DICT_OK;
    }