
### rehash过程
----dictscan函数分析

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
比特翻转，原理是每次都对半翻转高低位，一直到最后完成。
以8bit为例：
mask：1111 1111 
v： b1b2b3b4b5b6b7b8
第一步：
mask： 0000 1111
v：    b5b6b7b8 b1b2b3b4
第二步：
mask： 0011 0011
v:     b7b8b5b6 b3b4b1b2
第三步： 
mask:  0101 0101
v:     b8b7b6b5 b4b3b2b1

完成。


非rehash过程中

    v |= ~m0;假设v的低三位是x3x2x1，mask为00000111
和mask的补进行或，即是将mask的高位0都设置为1，低位为v的值。得到11111x3x2x1.
    v = rev(v);
将v的比特位进行反转，低位都变成1。得到x1x2x311111
    v++;
将v加1，因为v的低位都是1，所以会依次进位.得到(y1y2y3)00000,其中y1y2y3=x1x2x3+1
    v = rev(v);
再次进行比特位反转，得到00000y3y2y1.

相当于x3x2x1原先的高位是x3,现在把x3看成是低位，将其+1，即从左边进位，每次都是这样+1，

所以如果这个过程中没有发生resize的话，那么所有的槽位肯定都能走到。

从效果上来说，是每次都是变化的最高位。

从0开始上面的例子得到的结果为
00000000
00000100
00000010
00000110
00000001
00000101
00000011
00000111
过程中进行了扩容，size为原先的4倍，扩容后，只有低三位相同的槽位可能发生扩散。
所以这个变化高位的算法并不是为了匹配成对，对扩容来说，是为了保证低位相同的
数据不会被重复取到。
00000000
00000100
-----扩容----mask为00011111
00000010
00000110 这个地方变为00010010
	00001010
	00011010
这4个为1组，即低位为010，因为扩容之前没有取到低位为010的数据，所以不会有重复。
同时因为扩容后，高位总是从0开始，想象从左边进位+1，所以也不会漏数据。
而低位为000和100的，已经取过了。根据这个算法，低3位可以考虑为从左边进位的
高位，所以一旦过去后，就不会再取到了，因为相当于从左边进位+1，后面的数只会越来越大。
	00000110
	00000001

过程中进行了缩容，size为原先的1/2.
00000000
00000100
00000010
----缩容----mask为00000011
00000110----实际取00000010，所以会出现重复，（包括原先的00000010和00000110，而00000010已经取过一遍）
00000001
00000011
因为是高位变化，所以低位变化较慢。缩容后，如果之前已经取过的数据包含相同的低位，则会出现重复。
会重复的数据最多不会超过2的m次方，m为缩容前的logSize-缩容后的logSize.
又因为低位会继续变化，所以不会出现漏数据的情况。

过程中，先进行了扩容，又进行了缩容，也不会有遗漏数据。

整个过程都是无状态的。

rehash过程中，假设最小的表mask是00000111
从0开始V的值为：
00000000
00000100
00000010
00000110
00000001
00000101
00000011
00000111
如果循环处在一次rehash过程中，扫描到的数据不会重复也不会漏。

如果在循环过程中，rehash完成了（扩容）。

00000000
00000100
-----扩容----mask为00011111
00000010
00000110 这个地方变为00010010
00000001
00000101
00000011
00000111
分析过程和非rehash过程扩容一样。

如果在循环过程中，rehash完成了（缩容),size为原先的1/2.
00000000
00000100
00000010
----缩容----mask为00000011
00000110----实际取00000010，所以会出现重复，（包括原先的00000010和00000110，而00000010已经取过一遍）
00000001
00000011
分析过程和非rehash过程缩容一样。

如果在循环过程中，发生了多次rehash,并正在进行。
从0开始V的值为：
00000000
00000100
-----扩容----mask为00011111，这里是第二次rehash，并进行了扩容。
00000010
00000110 这个地方变为00010010
00000001
00000101
00000011
分析过程和非rehash过程扩容一样。

这个算法的关键是：
无论发生了多少次rehash，只要最后的size大小是一样，那么同一个键总是在同一个槽位上。
而在rehash的过程中，同一个键对应的槽位是有规律的。即扩容的时候，扩散到低位相同的槽位上，
缩容的时候，收缩到低位相同的槽位上，这个跟做了多少次rehash没有关系，只跟最后的size有关。
然后通过让高位变化，保证了扩容时，已经取到的元素不会重复取，同时缩容时不会丢失数据。
缩容时会重复的数据最多不会超过2的m次方，m为缩容前的logSize-缩容后的logSize.

unsigned long dictScan(dict *d,
                       unsigned long v,
                       dictScanFunction *fn,
                       void *privdata)
{
    dictht *t0, *t1;
    const dictEntry *de;
    unsigned long m0, m1;

    if (dictSize(d) == 0) return 0;

    if (!dictIsRehashing(d)) {
        t0 = &(d->ht[0]);
        m0 = t0->sizemask;

        /* Emit entries at cursor */
        de = t0->table[v & m0];
        while (de) {
            fn(privdata, de);
            de = de->next;
        }

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
        de = t0->table[v & m0];
        while (de) {
            fn(privdata, de);
            de = de->next;
        }

        /* Iterate over indices in larger table that are the expansion
         * of the index pointed to by the cursor in the smaller table */
        do {
            /* Emit entries at cursor */
            de = t1->table[v & m1];
            while (de) {
                fn(privdata, de);
                de = de->next;
            }

            /* Increment bits not covered by the smaller mask */
            v = (((v | m0) + 1) & ~m0) | (v & m0);

            /* Continue while bits covered by mask difference is non-zero */
        } while (v & (m0 ^ m1));
    }

    /* Set unmasked bits so incrementing the reversed cursor
     * operates on the masked bits of the smaller table */
    v |= ~m0;

    /* Increment the reverse cursor */
    v = rev(v);
    v++;
    v = rev(v);

    return v;
}


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



