## 列表是由ziplist来实现的

    /* The ziplist is a specially encoded dually linked list that is designed
    * to be very memory efficient. It stores both strings and integer values,
    * where integers are encoded as actual integers instead of a series of
    * characters. It allows push and pop operations on either side of the list
    * in O(1) time. However, because every operation requires a reallocation of
    * the memory used by the ziplist, the actual complexity is related to the
    * amount of memory used by the ziplist.
    * 这种实现方式比着普通链表实现方式节约的内存如下：
    * 普通列表：要有一个prev和next指针，这两个指针共占用16字节，而zipList前向是占用1个或5个字节，即prevlen，后向是占用1-5字节，即encoding。
    * ----------------------------------------------------------------------------
    *
    * ZIPLIST OVERALL LAYOUT
    * ======================
    *
    * The general layout of the ziplist is as follows:
    *
    * <zlbytes> <zltail> <zllen> <entry> <entry> ... <entry> <zlend>
    *
    * NOTE: all fields are stored in little endian, if not specified otherwise.
    *大端序（Big-Endian，大尾序）：高位字节放在内存的低地址，低位字节放在内存的高地址。小端序（Little-Endian，小尾序）：低位字节放在内存的低地址，高位字节放在内存的高地址。
    * <uint32_t zlbytes> is an unsigned integer to hold the number of bytes that
    * the ziplist occupies, including the four bytes of the zlbytes field itself.
    * This value needs to be stored to be able to resize the entire structure
    * without the need to traverse it first.
    *
    * <uint32_t zltail> is the offset to the last entry in the list. This allows
    * a pop operation on the far side of the list without the need for full
    * traversal.
    * 也就是说如果元素数量<65534,取列表长度复杂度为O(1),否则为O(N)。
    * <uint16_t zllen> is the number of entries. When there are more than
    * 2^16-2 entries, this value is set to 2^16-1 and we need to traverse the
    * entire list to know how many items it holds.
    *
    * <uint8_t zlend> is a special entry representing the end of the ziplist.
    * Is encoded as a single byte equal to 255. No other normal entry starts
    * with a byte set to the value of 255.
    *
    * ZIPLIST ENTRIES
    * ===============
    *
    * Every entry in the ziplist is prefixed by metadata that contains two pieces
    * of information. First, the length of the previous entry is stored to be
    * able to traverse the list from back to front. Second, the entry encoding is
    * provided. It represents the entry type, integer or string, and in the case
    * of strings it also represents the length of the string payload.
    * So a complete entry is stored like this:
    *
    * <prevlen> <encoding> <entry-data>
    *
    * Sometimes the encoding represents the entry itself, like for small integers
    * as we'll see later. In such a case the <entry-data> part is missing, and we
    * could have just:
    *
    * <prevlen> <encoding>
    *
    * The length of the previous entry, <prevlen>, is encoded in the following way:
    * If this length is smaller than 254 bytes, it will only consume a single
    * byte representing the length as an unsinged 8 bit integer. When the length
    * is greater than or equal to 254, it will consume 5 bytes. The first byte is
    * set to 254 (FE) to indicate a larger value is following. The remaining 4
    * bytes take the length of the previous entry as value.
    *
    * So practically an entry is encoded in the following way:
    *
    * <prevlen from 0 to 253> <encoding> <entry>
    *
    * Or alternatively if the previous entry length is greater than 253 bytes
    * the following encoding is used:
    *
    * 0xFE <4 bytes unsigned little endian prevlen> <encoding> <entry>
    *
    * The encoding field of the entry depends on the content of the
    * entry. When the entry is a string, the first 2 bits of the encoding first
    * byte will hold the type of encoding used to store the length of the string,
    * followed by the actual length of the string. When the entry is an integer
    * the first 2 bits are both set to 1. The following 2 bits are used to specify
    * what kind of integer will be stored after this header. An overview of the
    * different types and encodings is as follows. The first byte is always enough
    * to determine the kind of entry.
    *
    * |00pppppp| - 1 byte
    *      String value with length less than or equal to 63 bytes (6 bits).
    *      "pppppp" represents the unsigned 6 bit length.
    * |01pppppp|qqqqqqqq| - 2 bytes
    *      String value with length less than or equal to 16383 bytes (14 bits).
    *      IMPORTANT: The 14 bit number is stored in big endian.
    * |10000000|qqqqqqqq|rrrrrrrr|ssssssss|tttttttt| - 5 bytes
    *      String value with length greater than or equal to 16384 bytes.
    *      Only the 4 bytes following the first byte represents the length
    *      up to 2^32-1. The 6 lower bits of the first byte are not used and
    *      are set to zero.
    *      IMPORTANT: The 32 bit number is stored in big endian.
    * |11000000| - 3 bytes
    *      Integer encoded as int16_t (2 bytes).
    * |11010000| - 5 bytes
    *      Integer encoded as int32_t (4 bytes).
    * |11100000| - 9 bytes
    *      Integer encoded as int64_t (8 bytes).
    * |11110000| - 4 bytes
    *      Integer encoded as 24 bit signed (3 bytes).
    * |11111110| - 2 bytes
    *      Integer encoded as 8 bit signed (1 byte).
    * |1111xxxx| - (with xxxx between 0000 and 1101) immediate 4 bit integer.
    *      Unsigned integer from 0 to 12. The encoded value is actually from
    *      1 to 13 because 0000 and 1111 can not be used, so 1 should be
    *      subtracted from the encoded 4 bit value to obtain the right value.
    * |11111111| - End of ziplist special entry.
    *
    * Like for the ziplist header, all the integers are represented in little
    * endian byte order, even when this code is compiled in big endian systems.
    *
    * EXAMPLES OF ACTUAL ZIPLISTS
    * ===========================
    *
    * The following is a ziplist containing the two elements representing
    * the strings "2" and "5". It is composed of 15 bytes, that we visually
    * split into sections:
    *
    *  [0f 00 00 00] [0c 00 00 00] [02 00] [00 f3] [02 f6] [ff]
    *        |             |          |       |       |     |
    *     zlbytes        zltail    entries   "2"     "5"   end
    *
    * The first 4 bytes represent the number 15, that is the number of bytes
    * the whole ziplist is composed of. The second 4 bytes are the offset
    * at which the last ziplist entry is found, that is 12, in fact the
    * last entry, that is "5", is at offset 12 inside the ziplist.
    * The next 16 bit integer represents the number of elements inside the
    * ziplist, its value is 2 since there are just two elements inside.
    * Finally "00 f3" is the first entry representing the number 2. It is
    * composed of the previous entry length, which is zero because this is
    * our first entry, and the byte F3 which corresponds to the encoding
    * |1111xxxx| with xxxx between 0001 and 1101. We need to remove the "F"
    * higher order bits 1111, and subtract 1 from the "3", so the entry value
    * is "2". The next entry has a prevlen of 02, since the first entry is
    * composed of exactly two bytes. The entry itself, F6, is encoded exactly
    * like the first entry, and 6-1 = 5, so the value of the entry is 5.
    * Finally the special entry FF signals the end of the ziplist.
    *
    * Adding another element to the above string with the value "Hello World"
    * allows us to show how the ziplist encodes small strings. We'll just show
    * the hex dump of the entry itself. Imagine the bytes as following the
    * entry that stores "5" in the ziplist above:
    *
    * [02] [0b] [48 65 6c 6c 6f 20 57 6f 72 6c 64]
    *
    * The first byte, 02, is the length of the previous entry. The next
    * byte represents the encoding in the pattern |00pppppp| that means
    * that the entry is a string of length <pppppp>, so 0B means that
    * an 11 bytes string follows. From the third byte (48) to the last (64)
    * there are just the ASCII characters for "Hello World".
    */
    了解了上面介绍的数据结构，zipList的各种操作应该就比较好理解了。需要注意的是，每次zipList的操作，都会导致一次对zipList的realloc操作，因为zipList是一整块连续内存。

    /* Increment the number of items field in the ziplist header. Note that this
    * macro should never overflow the unsigned 16 bit integer, since entries are
    * always pushed one at a time. When UINT16_MAX is reached we want the count
    * to stay there to signal that a full scan is needed to get the number of
    * items inside the ziplist. */
    incr必须是1才行。
    #define ZIPLIST_INCR_LENGTH(zl,incr) { \
        if (ZIPLIST_LENGTH(zl) < UINT16_MAX) \
            ZIPLIST_LENGTH(zl) = intrev16ifbe(intrev16ifbe(ZIPLIST_LENGTH(zl))+incr); \
    }

    /* We use this function to receive information about a ziplist entry.
    * Note that this is not how the data is actually encoded, is just what we
    * get filled by a function in order to operate more easily. */
    typedef struct zlentry {
        unsigned int prevrawlensize; /* Bytes used to encode the previous entry len*/
        unsigned int prevrawlen;     /* Previous entry len. */
        unsigned int lensize;        /* Bytes used to encode this entry type/len.
                                        For example strings have a 1, 2 or 5 bytes
                                        header. Integers always use a single byte.*/
        unsigned int len;            /* Bytes used to represent the actual entry.
                                        For strings this is just the string length
                                        while for integers it is 1, 2, 3, 4, 8 or
                                        0 (for 4 bit immediate) depending on the
                                        number range. */
        unsigned int headersize;     /* prevrawlensize + lensize. */
        unsigned char encoding;      /* Set to ZIP_STR_* or ZIP_INT_* depending on
                                        the entry encoding. However for 4 bits
                                        immediate integers this can assume a range
                                        of values and must be range-checked. */
        unsigned char *p;            /* Pointer to the very start of the entry, that
                                        is, this points to prev-entry-len field. */
    } zlentry;

    #define ZIPLIST_ENTRY_ZERO(zle) { \
        (zle)->prevrawlensize = (zle)->prevrawlen = 0; \
        (zle)->lensize = (zle)->len = (zle)->headersize = 0; \
        (zle)->encoding = 0; \
        (zle)->p = NULL; \
    }

    /* Extract the encoding from the byte pointed by 'ptr' and set it into
    * 'encoding' field of the zlentry structure. */
    #define ZIP_ENTRY_ENCODING(ptr, encoding) do {  \
        (encoding) = (ptr[0]); \
        if ((encoding) < ZIP_STR_MASK) (encoding) &= ZIP_STR_MASK; \
    } while(0)

    /* Return bytes needed to store integer encoded by 'encoding'. */
    unsigned int zipIntSize(unsigned char encoding) {
        switch(encoding) {
        case ZIP_INT_8B:  return 1;
        case ZIP_INT_16B: return 2;
        case ZIP_INT_24B: return 3;
        case ZIP_INT_32B: return 4;
        case ZIP_INT_64B: return 8;
        }
        if (encoding >= ZIP_INT_IMM_MIN && encoding <= ZIP_INT_IMM_MAX)
            return 0; /* 4 bit immediate */
        panic("Invalid integer encoding 0x%02X", encoding);
        return 0;
    }

    /* Write the encoidng header of the entry in 'p'. If p is NULL it just returns
    * the amount of bytes required to encode such a length. Arguments:
    *
    * 'encoding' is the encoding we are using for the entry. It could be
    * ZIP_INT_* or ZIP_STR_* or between ZIP_INT_IMM_MIN and ZIP_INT_IMM_MAX
    * for single-byte small immediate integers.
    *
    * 'rawlen' is only used for ZIP_STR_* encodings and is the length of the
    * srting that this entry represents.
    *
    * The function returns the number of bytes used by the encoding/length
    * header stored in 'p'. */
    unsigned int zipStoreEntryEncoding(unsigned char *p, unsigned char encoding, unsigned int rawlen) {
        unsigned char len = 1, buf[5];

        if (ZIP_IS_STR(encoding)) {
            /* Although encoding is given it may not be set for strings,
            * so we determine it here using the raw length. */
            if (rawlen <= 0x3f) {
                if (!p) return len;
                buf[0] = ZIP_STR_06B | rawlen;
            } else if (rawlen <= 0x3fff) {
                len += 1;
                if (!p) return len;
                buf[0] = ZIP_STR_14B | ((rawlen >> 8) & 0x3f);
                buf[1] = rawlen & 0xff;
            } else {
                len += 4;
                if (!p) return len;
                buf[0] = ZIP_STR_32B;
                下面这个长度为大端序。
                buf[1] = (rawlen >> 24) & 0xff;
                buf[2] = (rawlen >> 16) & 0xff;
                buf[3] = (rawlen >> 8) & 0xff;
                buf[4] = rawlen & 0xff;
            }
        } else {
            /* Implies integer encoding, so length is always 1. */
            if (!p) return len;
            buf[0] = encoding;
        }

        /* Store this length at p. */
        memcpy(p,buf,len);
        return len;
    }

    /* Decode the entry encoding type and data length (string length for strings,
    * number of bytes used for the integer for integer entries) encoded in 'ptr'.
    * The 'encoding' variable will hold the entry encoding, the 'lensize'
    * variable will hold the number of bytes required to encode the entry
    * length, and the 'len' variable will hold the entry length. */
    从这里可以看到使用宏可以直接使用变量而不是指针来设置值。如果有时候看到设置了变量，那么实际上不是使用的函数而是宏。
    #define ZIP_DECODE_LENGTH(ptr, encoding, lensize, len) do {                    \
        ZIP_ENTRY_ENCODING((ptr), (encoding));                                     \
        if ((encoding) < ZIP_STR_MASK) {                                           \
            if ((encoding) == ZIP_STR_06B) {                                       \
                (lensize) = 1;                                                     \
                (len) = (ptr)[0] & 0x3f;                                           \
            } else if ((encoding) == ZIP_STR_14B) {                                \
                (lensize) = 2;                                                     \
                (len) = (((ptr)[0] & 0x3f) << 8) | (ptr)[1];                       \
            } else if ((encoding) == ZIP_STR_32B) {                                \
                (lensize) = 5;                                                     \
                (len) = ((ptr)[1] << 24) |                                         \
                        ((ptr)[2] << 16) |                                         \
                        ((ptr)[3] <<  8) |                                         \
                        ((ptr)[4]);                                                \
            } else {                                                               \
                panic("Invalid string encoding 0x%02X", (encoding));               \
            }                                                                      \
        } else {                                                                   \
            (lensize) = 1;                                                         \
            (len) = zipIntSize(encoding);                                          \
        }                                                                          \
    } while(0);

    /* Encode the length of the previous entry and write it to "p". This only
    * uses the larger encoding (required in __ziplistCascadeUpdate). */
    int zipStorePrevEntryLengthLarge(unsigned char *p, unsigned int len) {
        if (p != NULL) {
            p[0] = ZIP_BIG_PREVLEN;
            memcpy(p+1,&len,sizeof(len));
            memrev32ifbe(p+1);
        }
        return 1+sizeof(len);
    }

    /* Encode the length of the previous entry and write it to "p". Return the
    * number of bytes needed to encode this length if "p" is NULL. */
    unsigned int zipStorePrevEntryLength(unsigned char *p, unsigned int len) {
        if (p == NULL) {
            return (len < ZIP_BIG_PREVLEN) ? 1 : sizeof(len)+1;
        } else {
            if (len < ZIP_BIG_PREVLEN) {
                p[0] = len;
                return 1;
            } else {
                return zipStorePrevEntryLengthLarge(p,len);
            }
        }
    }

    /* Return the number of bytes used to encode the length of the previous
    * entry. The length is returned by setting the var 'prevlensize'. */
    #define ZIP_DECODE_PREVLENSIZE(ptr, prevlensize) do {                          \
        if ((ptr)[0] < ZIP_BIG_PREVLEN) {                                          \
            (prevlensize) = 1;                                                     \
        } else {                                                                   \
            (prevlensize) = 5;                                                     \
        }                                                                          \
    } while(0);

    /* Return the length of the previous element, and the number of bytes that
    * are used in order to encode the previous element length.
    * 'ptr' must point to the prevlen prefix of an entry (that encodes the
    * length of the previous entry in order to navigate the elements backward).
    * The length of the previous entry is stored in 'prevlen', the number of
    * bytes needed to encode the previous entry length are stored in
    * 'prevlensize'. */
    #define ZIP_DECODE_PREVLEN(ptr, prevlensize, prevlen) do {                     \
        ZIP_DECODE_PREVLENSIZE(ptr, prevlensize);                                  \
        if ((prevlensize) == 1) {                                                  \
            (prevlen) = (ptr)[0];                                                  \
        } else if ((prevlensize) == 5) {                                           \
            assert(sizeof((prevlen)) == 4);                                    \
            memcpy(&(prevlen), ((char*)(ptr)) + 1, 4);                             \
            memrev32ifbe(&prevlen);                                                \
        }                                                                          \
    } while(0);

    /* Given a pointer 'p' to the prevlen info that prefixes an entry, this
    * function returns the difference in number of bytes needed to encode
    * the prevlen if the previous entry changes of size.
    *
    * So if A is the number of bytes used right now to encode the 'prevlen'
    * field.
    *
    * And B is the number of bytes that are needed in order to encode the
    * 'prevlen' if the previous element will be updated to one of size 'len'.
    *
    * Then the function returns B - A
    *
    * So the function returns a positive number if more space is needed,
    * a negative number if less space is needed, or zero if the same space
    * is needed. */
    int zipPrevLenByteDiff(unsigned char *p, unsigned int len) {
        unsigned int prevlensize;
        ZIP_DECODE_PREVLENSIZE(p, prevlensize);
        return zipStorePrevEntryLength(NULL, len) - prevlensize;
    }

    /* Return the total number of bytes used by the entry pointed to by 'p'. */
    unsigned int zipRawEntryLength(unsigned char *p) {
        unsigned int prevlensize, encoding, lensize, len;
        ZIP_DECODE_PREVLENSIZE(p, prevlensize);
        ZIP_DECODE_LENGTH(p + prevlensize, encoding, lensize, len);
        return prevlensize + lensize + len;
    }

    /* Check if string pointed to by 'entry' can be encoded as an integer.
    * Stores the integer value in 'v' and its encoding in 'encoding'. */
    int zipTryEncoding(unsigned char *entry, unsigned int entrylen, long long *v, unsigned char *encoding) {
        long long value;

        if (entrylen >= 32 || entrylen == 0) return 0;
        if (string2ll((char*)entry,entrylen,&value)) {
            /* Great, the string can be encoded. Check what's the smallest
            * of our encoding types that can hold this value. */
            if (value >= 0 && value <= 12) {
                *encoding = ZIP_INT_IMM_MIN+value;
            } else if (value >= INT8_MIN && value <= INT8_MAX) {
                *encoding = ZIP_INT_8B;
            } else if (value >= INT16_MIN && value <= INT16_MAX) {
                *encoding = ZIP_INT_16B;
            } else if (value >= INT24_MIN && value <= INT24_MAX) {
                *encoding = ZIP_INT_24B;
            } else if (value >= INT32_MIN && value <= INT32_MAX) {
                *encoding = ZIP_INT_32B;
            } else {
                *encoding = ZIP_INT_64B;
            }
            *v = value;
            return 1;
        }
        return 0;
    }

    /* Store integer 'value' at 'p', encoded as 'encoding' */
    void zipSaveInteger(unsigned char *p, int64_t value, unsigned char encoding) {
        int16_t i16;
        int32_t i32;
        int64_t i64;
        if (encoding == ZIP_INT_8B) {
            ((int8_t*)p)[0] = (int8_t)value;
        } else if (encoding == ZIP_INT_16B) {
            i16 = value;
            memcpy(p,&i16,sizeof(i16));
            memrev16ifbe(p);
        } else if (encoding == ZIP_INT_24B) {
            i32 = value<<8;
            memrev32ifbe(&i32);
            memcpy(p,((uint8_t*)&i32)+1,sizeof(i32)-sizeof(uint8_t));
        } else if (encoding == ZIP_INT_32B) {
            i32 = value;
            memcpy(p,&i32,sizeof(i32));
            memrev32ifbe(p);
        } else if (encoding == ZIP_INT_64B) {
            i64 = value;
            memcpy(p,&i64,sizeof(i64));
            memrev64ifbe(p);
        } else if (encoding >= ZIP_INT_IMM_MIN && encoding <= ZIP_INT_IMM_MAX) {
            /* Nothing to do, the value is stored in the encoding itself. */
        } else {
            assert(NULL);
        }
    }

    /* Read integer encoded as 'encoding' from 'p' */
    int64_t zipLoadInteger(unsigned char *p, unsigned char encoding) {
        int16_t i16;
        int32_t i32;
        int64_t i64, ret = 0;
        if (encoding == ZIP_INT_8B) {
            ret = ((int8_t*)p)[0];
        } else if (encoding == ZIP_INT_16B) {
            memcpy(&i16,p,sizeof(i16));
            memrev16ifbe(&i16);
            ret = i16;
        } else if (encoding == ZIP_INT_32B) {
            memcpy(&i32,p,sizeof(i32));
            memrev32ifbe(&i32);
            ret = i32;
        } else if (encoding == ZIP_INT_24B) {
            i32 = 0;
            memcpy(((uint8_t*)&i32)+1,p,sizeof(i32)-sizeof(uint8_t));
            memrev32ifbe(&i32);
            ret = i32>>8;
        } else if (encoding == ZIP_INT_64B) {
            memcpy(&i64,p,sizeof(i64));
            memrev64ifbe(&i64);
            ret = i64;
        } else if (encoding >= ZIP_INT_IMM_MIN && encoding <= ZIP_INT_IMM_MAX) {
            ret = (encoding & ZIP_INT_IMM_MASK)-1;
        } else {
            assert(NULL);
        }
        return ret;
    }

    /* Return a struct with all information about an entry. */
    void zipEntry(unsigned char *p, zlentry *e) {

        ZIP_DECODE_PREVLEN(p, e->prevrawlensize, e->prevrawlen);
        ZIP_DECODE_LENGTH(p + e->prevrawlensize, e->encoding, e->lensize, e->len);
        e->headersize = e->prevrawlensize + e->lensize;
        e->p = p;
    }

    /* Create a new empty ziplist. */
    unsigned char *ziplistNew(void) {
        unsigned int bytes = ZIPLIST_HEADER_SIZE+1;
        unsigned char *zl = zmalloc(bytes);
        ZIPLIST_BYTES(zl) = intrev32ifbe(bytes);
        ZIPLIST_TAIL_OFFSET(zl) = intrev32ifbe(ZIPLIST_HEADER_SIZE);
        ZIPLIST_LENGTH(zl) = 0;
        zl[bytes-1] = ZIP_END;
        return zl;
    }

    /* Resize the ziplist. */
    unsigned char *ziplistResize(unsigned char *zl, unsigned int len) {
        zl = zrealloc(zl,len);
        ZIPLIST_BYTES(zl) = intrev32ifbe(len);
        zl[len-1] = ZIP_END;
        return zl;
    }

    处理级联更新prevlen，大批量的级联概率很低。该方法执行的时候，元素已经插入了。
    /* When an entry is inserted, we need to set the prevlen field of the next
    * entry to equal the length of the inserted entry. It can occur that this
    * length cannot be encoded in 1 byte and the next entry needs to be grow
    * a bit larger to hold the 5-byte encoded prevlen. This can be done for free,
    * because this only happens when an entry is already being inserted (which
    * causes a realloc and memmove). However, encoding the prevlen may require
    * that this entry is grown as well. This effect may cascade throughout
    * the ziplist when there are consecutive entries with a size close to
    * ZIP_BIG_PREVLEN, so we need to check that the prevlen can be encoded in
    * every consecutive entry.
    * 也就是说在shrink的时候，prevlen可以仍然保持5字节。长度是通过encoding和后面的字节共同决定的。 
    * Note that this effect can also happen in reverse, where the bytes required
    * to encode the prevlen field can shrink. This effect is deliberately ignored,
    * because it can cause a "flapping" effect where a chain prevlen fields is
    * first grown and then shrunk again after consecutive inserts. Rather, the
    * field is allowed to stay larger than necessary, because a large prevlen
    * field implies the ziplist is holding large entries anyway.
    *
    * The pointer "p" points to the first entry that does NOT need to be
    * updated, i.e. consecutive fields MAY need an update. */
    p指的是prev元素本身。
    unsigned char *__ziplistCascadeUpdate(unsigned char *zl, unsigned char *p) {
        size_t curlen = intrev32ifbe(ZIPLIST_BYTES(zl)), rawlen, rawlensize;
        size_t offset, noffset, extra;
        unsigned char *np;
        zlentry cur, next;

        while (p[0] != ZIP_END) {
            zipEntry(p, &cur);
            rawlen = cur.headersize + cur.len;
            rawlensize = zipStorePrevEntryLength(NULL,rawlen);

            /* Abort if there is no next entry. */
            if (p[rawlen] == ZIP_END) break;
            zipEntry(p+rawlen, &next);

            /* Abort when "prevlen" has not changed. */
            if (next.prevrawlen == rawlen) break;
            注意这里判断的是prevrawlensize，而不是prevlen本身，即用来存储prevlen的字节数，1或5字节。
            if (next.prevrawlensize < rawlensize) {
                /* The "prevlen" field of "next" needs more bytes to hold
                * the raw length of "cur". */
                offset = p-zl;
                extra = rawlensize-next.prevrawlensize;
                扩展ziplist长度。   
                zl = ziplistResize(zl,curlen+extra);
                p在新的ziplist中的位置。
                p = zl+offset; 

                /* Current pointer and offset for next element. */
                np = p+rawlen;
                next元素在新的zipList中新的位置。
                noffset = np-zl; 

                /* Update tail offset when next element is not the tail element. */
                if ((zl+intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))) != np) {
                    ZIPLIST_TAIL_OFFSET(zl) =
                        intrev32ifbe(intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))+extra);
                }

                /* Move the tail to the back. */
                将np+next.prevrawlensize位置开始的元素复制到np+rawlensize。
                在realloc之后虽然可用内存多了，但是内存布局并没有变，所以需要copy的内存数量为curlen-1(当前有效内存的最大偏移量) - noffset(np的偏移量) - next.prevrawlensize(之前的prevlensize)。该位置开始的内存即为next除了prevlen之后的内容。
                memmove(np+rawlensize,
                    np+next.prevrawlensize,
                    curlen-noffset-next.prevrawlensize-1);
                zipStorePrevEntryLength(np,rawlen);

                /* Advance the cursor */
                p += rawlen;
                curlen += extra;
            } else {
                if (next.prevrawlensize > rawlensize) {
                    /* This would result in shrinking, which we want to avoid.
                    * So, set "rawlen" in the available bytes. */
                    zipStorePrevEntryLengthLarge(p+rawlen,rawlen);
                } else {
                    zipStorePrevEntryLength(p+rawlen,rawlen);
                }

                /* Stop here, as the raw length of "next" has not changed. */
                break;
            }
        }
        return zl;
    }

    /* Delete "num" entries, starting at "p". Returns pointer to the ziplist. */
    unsigned char *__ziplistDelete(unsigned char *zl, unsigned char *p, unsigned int num) {
        unsigned int i, totlen, deleted = 0;
        size_t offset;
        int nextdiff = 0;
        zlentry first, tail;

        zipEntry(p, &first);
        for (i = 0; p[0] != ZIP_END && i < num; i++) {
            p += zipRawEntryLength(p);
            deleted++;
        }

        totlen = p-first.p; /* Bytes taken by the element(s) to delete. */
        if (totlen > 0) {
            if (p[0] != ZIP_END) {
                /* Storing `prevrawlen` in this entry may increase or decrease the
                * number of bytes required compare to the current `prevrawlen`.
                * There always is room to store this, because it was previously
                * stored by an entry that is now being deleted. */
                差值可能为正也可能为负。
                nextdiff = zipPrevLenByteDiff(p,first.prevrawlen);

                /* Note that there is always space when p jumps backward: if
                * the new previous entry is large, one of the deleted elements
                * had a 5 bytes prevlen header, so there is for sure at least
                * 5 bytes free and we need just 4. */
                重新设置p的位置，并重置prevlen。
                p -= nextdiff;
                zipStorePrevEntryLength(p,first.prevrawlen);

                /* Update offset for tail */
                ZIPLIST_TAIL_OFFSET(zl) =
                    intrev32ifbe(intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))-totlen);

                /* When the tail contains more than one entry, we need to take
                * "nextdiff" in account as well. Otherwise, a change in the
                * size of prevlen doesn't have an effect on the *tail* offset. */
                zipEntry(p, &tail);
                如果p是tail则tail offset即为原先的tail offset - 要删除的字节数，即first的位置；否则tail offset为原先的tail offset - 要删除的字节数 + prevlensize的差值。 这里的意思是如果p为tail，那么prevlen的变化不会改变p的offset，因为prevlen是在p的offset后面变化；而如果p不是tail，那么prevlen有可能会改变tail的offset，因为是在tail的offset前面变化。
                if (p[tail.headersize+tail.len] != ZIP_END) {
                    ZIPLIST_TAIL_OFFSET(zl) =
                    intrev32ifbe(intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))+nextdiff);
                }

                /* Move tail to the front of the ziplist */
                memmove(first.p,p,
                    intrev32ifbe(ZIPLIST_BYTES(zl))-(p-zl)-1);
            } else {
                /* The entire tail was deleted. No need to move memory. */
                如果删除的是直到tail的list，那么tail就变成了first的prev.
                ZIPLIST_TAIL_OFFSET(zl) =
                    intrev32ifbe((first.p-zl)-first.prevrawlen);
            }

            /* Resize and update length */
            offset = first.p-zl;
            zl = ziplistResize(zl, intrev32ifbe(ZIPLIST_BYTES(zl))-totlen+nextdiff);
            ZIPLIST_INCR_LENGTH(zl,-deleted);
            p = zl+offset;

            /* When nextdiff != 0, the raw length of the next entry has changed, so
            * we need to cascade the update throughout the ziplist */
            if (nextdiff != 0)
                zl = __ziplistCascadeUpdate(zl,p);
        }
        return zl;
    }

    /* Insert item at "p". */
    unsigned char *__ziplistInsert(unsigned char *zl, unsigned char *p, unsigned char *s, unsigned int slen) {
        size_t curlen = intrev32ifbe(ZIPLIST_BYTES(zl)), reqlen;
        unsigned int prevlensize, prevlen = 0;
        size_t offset;
        int nextdiff = 0;
        unsigned char encoding = 0;
        long long value = 123456789; /* initialized to avoid warning. Using a value
                                        that is easy to see if for some reason
                                        we use it uninitialized. */
        zlentry tail;
       
        /* Find out prevlen for the entry that is inserted. */
        if (p[0] != ZIP_END) {
            ZIP_DECODE_PREVLEN(p, prevlensize, prevlen);
        } else {
            p为ZIP_END，即在末尾插入。
            unsigned char *ptail = ZIPLIST_ENTRY_TAIL(zl);
            // 如果不是空的列表。
            if (ptail[0] != ZIP_END) {
                prevlen = zipRawEntryLength(ptail);
            }
        }

        /* See if the entry can be encoded */
         如果可以将字符串作为整数存储，那么将其作为整数存储。
        if (zipTryEncoding(s,slen,&value,&encoding)) {
            /* 'encoding' is set to the appropriate integer encoding */
            reqlen = zipIntSize(encoding);
        } else {
            /* 'encoding' is untouched, however zipStoreEntryEncoding will use the
            * string length to figure out how to encode it. */
            reqlen = slen;
        }
        /* We need space for both the length of the previous entry and
        * the length of the payload. */
        reqlen += zipStorePrevEntryLength(NULL,prevlen);
        reqlen += zipStoreEntryEncoding(NULL,encoding,slen);

        /* When the insert position is not equal to the tail, we need to
        * make sure that the next entry can hold this entry's length in
        * its prevlen field. */
        int forcelarge = 0;
        nextdiff = (p[0] != ZIP_END) ? zipPrevLenByteDiff(p,reqlen) : 0;
    这个forcelarge在之前的版本是没有的，在2017.1.30加入进来的，说明如下：
    Ziplist: insertion bug under particular conditions fixed.
    Ziplists had a bug that was discovered while investigating a different
    issue, resulting in a corrupted ziplist representation, and a likely
    segmentation foult and/or data corruption of the last element of the
    ziplist, once the ziplist is accessed again.

    The bug happens when a specific set of insertions / deletions is
    performed so that an entry is encoded to have a "prevlen" field (the
    length of the previous entry) of 5 bytes but with a count that could be
    encoded in a "prevlen" field of a since byte. This could happen when the
    "cascading update" process called by ziplistInsert()/ziplistDelete() in
    certain contitious forces the prevlen to be bigger than necessary in
    order to avoid too much data moving around.

    Once such an entry is generated, inserting a very small entry
    immediately before it will result in a resizing of the ziplist for a
    count smaller than the current ziplist length (which is a violation,
    inserting code expects the ziplist to get bigger actually). So an FF
    byte is inserted in a misplaced position. Moreover a realloc() is
    performed with a count smaller than the ziplist current length so the
    final bytes could be trashed as well.

    SECURITY IMPLICATIONS:

    Currently it looks like an attacker can only crash a Redis server by
    providing specifically choosen commands. However a FF byte is written
    and there are other memory operations that depend on a wrong count, so
    even if it is not immediately apparent how to mount an attack in order
    to execute code remotely, it is not impossible at all that this could be
    done. Attacks always get better... and we did not spent enough time in
    order to think how to exploit this issue, but security researchers
    or malicious attackers could.
    我们来看一下，如果不使用forcelarge，是怎样产生bug的。 首先nextdiff == -4,
        if (nextdiff == -4 && reqlen < 4) {
            nextdiff = 0;
            forcelarge = 1;
        }

        /* Store offset because a realloc may change the address of zl. */
        offset = p-zl;
        // 不使用forcelarge，这里resize之后内存实际是缩小了，因为reqlen<4,nextdiff=-4.这里实际就丢数据了。因为多余的数据是在前面存储prev的5个字节中多出的4个，但是realloc是截断了后面的。所以在这种情况下，需要forcelarge，将nextdiff设为0，实际就多出了4字节，这样就不会多截断数据。从理论上来说，即使不多出这4字节也是可以实现的。就是先memmove，再realloc，关键是要认识到，虽然是插入数据，但是整体的数据量是变小了。
        zl = ziplistResize(zl,curlen+reqlen+nextdiff);
        p = zl+offset;

        /* Apply memory move when necessary and update tail offset. */
        if (p[0] != ZIP_END) {
            /* Subtract one because of the ZIP_END bytes */
            从p到p+reqlen为新加的元素，p-nextdiff表示copy的内存从p减去reqlen需要的prevlensize和p本身的prevlensize的差，即假设新的prevlensize为1，老的prevlensize为5，那么从p+4开始复制，丢掉4个字节；如果新的prevlensize为5，老的prevlensize为1，那么从p-4开始复制，多保留4个字节用来存储prevlen。复制的长度中使用nextdiff的方式正好相反。
            在forcelarge情况下，nextdiff为0，所以是从p开始，将内存移到p+reqlen位置。
            memmove(p+reqlen,p-nextdiff,curlen-offset-1+nextdiff);

            /* Encode this entry's raw length in the next entry. */
            if (forcelarge)
                zipStorePrevEntryLengthLarge(p+reqlen,reqlen);
            else
                zipStorePrevEntryLength(p+reqlen,reqlen);

            /* Update offset for tail */
            ZIPLIST_TAIL_OFFSET(zl) =
                intrev32ifbe(intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))+reqlen);

            /* When the tail contains more than one entry, we need to take
            * "nextdiff" in account as well. Otherwise, a change in the
            * size of prevlen doesn't have an effect on the *tail* offset. */
            zipEntry(p+reqlen, &tail);
            if (p[reqlen+tail.headersize+tail.len] != ZIP_END) {
                ZIPLIST_TAIL_OFFSET(zl) =
                    intrev32ifbe(intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl))+nextdiff);
            }
        } else {
            /* This element will be the new tail. */
            ZIPLIST_TAIL_OFFSET(zl) = intrev32ifbe(p-zl);
        }

        /* When nextdiff != 0, the raw length of the next entry has changed, so
        * we need to cascade the update throughout the ziplist */
        if (nextdiff != 0) {
            offset = p-zl;
            zl = __ziplistCascadeUpdate(zl,p+reqlen);
            p = zl+offset;
        }

        /* Write the entry */
        p += zipStorePrevEntryLength(p,prevlen);
        p += zipStoreEntryEncoding(p,encoding,slen);
        if (ZIP_IS_STR(encoding)) {
            memcpy(p,s,slen);
        } else {
            zipSaveInteger(p,value,encoding);
        }
        ZIPLIST_INCR_LENGTH(zl,1);
        return zl;
    }

    /* Merge ziplists 'first' and 'second' by appending 'second' to 'first'.
    *
    * NOTE: The larger ziplist is reallocated to contain the new merged ziplist.
    * Either 'first' or 'second' can be used for the result.  The parameter not
    * used will be free'd and set to NULL.
    *
    * After calling this function, the input parameters are no longer valid since
    * they are changed and free'd in-place.
    *
    * The result ziplist is the contents of 'first' followed by 'second'.
    *
    * On failure: returns NULL if the merge is impossible.
    * On success: returns the merged ziplist (which is expanded version of either
    * 'first' or 'second', also frees the other unused input ziplist, and sets the
    * input ziplist argument equal to newly reallocated ziplist return value. */
    unsigned char *ziplistMerge(unsigned char **first, unsigned char **second) {
        /* If any params are null, we can't merge, so NULL. */
        if (first == NULL || *first == NULL || second == NULL || *second == NULL)
            return NULL;

        /* Can't merge same list into itself. */
        if (*first == *second)
            return NULL;

        size_t first_bytes = intrev32ifbe(ZIPLIST_BYTES(*first));
        size_t first_len = intrev16ifbe(ZIPLIST_LENGTH(*first));

        size_t second_bytes = intrev32ifbe(ZIPLIST_BYTES(*second));
        size_t second_len = intrev16ifbe(ZIPLIST_LENGTH(*second));

        int append;
        unsigned char *source, *target;
        size_t target_bytes, source_bytes;
        /* Pick the largest ziplist so we can resize easily in-place.
        * We must also track if we are now appending or prepending to
        * the target ziplist. */
        if (first_len >= second_len) {
            /* retain first, append second to first. */
            target = *first;
            target_bytes = first_bytes;
            source = *second;
            source_bytes = second_bytes;
            append = 1;
        } else {
            /* else, retain second, prepend first to second. */
            target = *second;
            target_bytes = second_bytes;
            source = *first;
            source_bytes = first_bytes;
            append = 0;
        }

        /* Calculate final bytes (subtract one pair of metadata) */
        size_t zlbytes = first_bytes + second_bytes -
                        ZIPLIST_HEADER_SIZE - ZIPLIST_END_SIZE;
        size_t zllength = first_len + second_len;

        /* Combined zl length should be limited within UINT16_MAX */
        zllength = zllength < UINT16_MAX ? zllength : UINT16_MAX;

        /* Save offset positions before we start ripping memory apart. */
        size_t first_offset = intrev32ifbe(ZIPLIST_TAIL_OFFSET(*first));
        size_t second_offset = intrev32ifbe(ZIPLIST_TAIL_OFFSET(*second));

        /* Extend target to new zlbytes then append or prepend source. */
        target = zrealloc(target, zlbytes);
        if (append) {
            /* append == appending to target */
            /* Copy source after target (copying over original [END]):
            *   [TARGET - END, SOURCE - HEADER] */
            memcpy(target + target_bytes - ZIPLIST_END_SIZE,
                source + ZIPLIST_HEADER_SIZE,
                source_bytes - ZIPLIST_HEADER_SIZE);
        } else {
            /* !append == prepending to target */
            /* Move target *contents* exactly size of (source - [END]),
            * then copy source into vacataed space (source - [END]):
            *   [SOURCE - END, TARGET - HEADER] */
            memmove(target + source_bytes - ZIPLIST_END_SIZE,
                    target + ZIPLIST_HEADER_SIZE,
                    target_bytes - ZIPLIST_HEADER_SIZE);
            memcpy(target, source, source_bytes - ZIPLIST_END_SIZE);
        }

        /* Update header metadata. */
        ZIPLIST_BYTES(target) = intrev32ifbe(zlbytes);
        ZIPLIST_LENGTH(target) = intrev16ifbe(zllength);
        /* New tail offset is:
        *   + N bytes of first ziplist
        *   - 1 byte for [END] of first ziplist
        *   + M bytes for the offset of the original tail of the second ziplist
        *   - J bytes for HEADER because second_offset keeps no header. */
        ZIPLIST_TAIL_OFFSET(target) = intrev32ifbe(
                                    (first_bytes - ZIPLIST_END_SIZE) +
                                    (second_offset - ZIPLIST_HEADER_SIZE));

        /* __ziplistCascadeUpdate just fixes the prev length values until it finds a
        * correct prev length value (then it assumes the rest of the list is okay).
        * We tell CascadeUpdate to start at the first ziplist's tail element to fix
        * the merge seam. */
        target = __ziplistCascadeUpdate(target, target+first_offset);

        /* Now free and NULL out what we didn't realloc */
        if (append) {
            zfree(*second);
            *second = NULL;
            *first = target;
        } else {
            zfree(*first);
            *first = NULL;
            *second = target;
        }
        return target;
    }

    unsigned char *ziplistPush(unsigned char *zl, unsigned char *s, unsigned int slen, int where) {
        unsigned char *p;
        p = (where == ZIPLIST_HEAD) ? ZIPLIST_ENTRY_HEAD(zl) : ZIPLIST_ENTRY_END(zl);
        return __ziplistInsert(zl,p,s,slen);
    }

    /* Returns an offset to use for iterating with ziplistNext. When the given
    * index is negative, the list is traversed back to front. When the list
    * doesn't contain an element at the provided index, NULL is returned. */
    unsigned char *ziplistIndex(unsigned char *zl, int index) {
        unsigned char *p;
        unsigned int prevlensize, prevlen = 0;
        if (index < 0) {
            index = (-index)-1;
            p = ZIPLIST_ENTRY_TAIL(zl);
            if (p[0] != ZIP_END) {
                ZIP_DECODE_PREVLEN(p, prevlensize, prevlen);
                while (prevlen > 0 && index--) {
                    p -= prevlen;
                    ZIP_DECODE_PREVLEN(p, prevlensize, prevlen);
                }
            }
        } else {
            p = ZIPLIST_ENTRY_HEAD(zl);
            while (p[0] != ZIP_END && index--) {
                p += zipRawEntryLength(p);
            }
        }
        return (p[0] == ZIP_END || index > 0) ? NULL : p;
    }

    /* Return pointer to next entry in ziplist.
    *
    * zl is the pointer to the ziplist
    * p is the pointer to the current element
    *
    * The element after 'p' is returned, otherwise NULL if we are at the end. */
    unsigned char *ziplistNext(unsigned char *zl, unsigned char *p) {
        ((void) zl);

        /* "p" could be equal to ZIP_END, caused by ziplistDelete,
        * and we should return NULL. Otherwise, we should return NULL
        * when the *next* element is ZIP_END (there is no next entry). */
        if (p[0] == ZIP_END) {
            return NULL;
        }

        p += zipRawEntryLength(p);
        if (p[0] == ZIP_END) {
            return NULL;
        }

        return p;
    }

    /* Return pointer to previous entry in ziplist. */
    unsigned char *ziplistPrev(unsigned char *zl, unsigned char *p) {
        unsigned int prevlensize, prevlen = 0;

        /* Iterating backwards from ZIP_END should return the tail. When "p" is
        * equal to the first element of the list, we're already at the head,
        * and should return NULL. */
        if (p[0] == ZIP_END) {
            p = ZIPLIST_ENTRY_TAIL(zl);
            return (p[0] == ZIP_END) ? NULL : p;
        } else if (p == ZIPLIST_ENTRY_HEAD(zl)) {
            return NULL;
        } else {
            ZIP_DECODE_PREVLEN(p, prevlensize, prevlen);
            assert(prevlen > 0);
            return p-prevlen;
        }
    }

    /* Get entry pointed to by 'p' and store in either '*sstr' or 'sval' depending
    * on the encoding of the entry. '*sstr' is always set to NULL to be able
    * to find out whether the string pointer or the integer value was set.
    * Return 0 if 'p' points to the end of the ziplist, 1 otherwise. */
    unsigned int ziplistGet(unsigned char *p, unsigned char **sstr, unsigned int *slen, long long *sval) {
        zlentry entry;
        if (p == NULL || p[0] == ZIP_END) return 0;
        if (sstr) *sstr = NULL;

        zipEntry(p, &entry);
        if (ZIP_IS_STR(entry.encoding)) {
            if (sstr) {
                *slen = entry.len;
                *sstr = p+entry.headersize;
            }
        } else {
            if (sval) {
                *sval = zipLoadInteger(p+entry.headersize,entry.encoding);
            }
        }
        return 1;
    }

    /* Insert an entry at "p". */
    unsigned char *ziplistInsert(unsigned char *zl, unsigned char *p, unsigned char *s, unsigned int slen) {
        return __ziplistInsert(zl,p,s,slen);
    }

    /* Delete a single entry from the ziplist, pointed to by *p.
    * Also update *p in place, to be able to iterate over the
    * ziplist, while deleting entries. */
    unsigned char *ziplistDelete(unsigned char *zl, unsigned char **p) {
        size_t offset = *p-zl;
        zl = __ziplistDelete(zl,*p,1);

        /* Store pointer to current element in p, because ziplistDelete will
        * do a realloc which might result in a different "zl"-pointer.
        * When the delete direction is back to front, we might delete the last
        * entry and end up with "p" pointing to ZIP_END, so check this. */
        *p = zl+offset;
        return zl;
    }

    /* Delete a range of entries from the ziplist. */
    unsigned char *ziplistDeleteRange(unsigned char *zl, int index, unsigned int num) {
        unsigned char *p = ziplistIndex(zl,index);
        return (p == NULL) ? zl : __ziplistDelete(zl,p,num);
    }

    /* Compare entry pointer to by 'p' with 'sstr' of length 'slen'. */
    /* Return 1 if equal. */
    unsigned int ziplistCompare(unsigned char *p, unsigned char *sstr, unsigned int slen) {
        zlentry entry;
        unsigned char sencoding;
        long long zval, sval;
        if (p[0] == ZIP_END) return 0;

        zipEntry(p, &entry);
        if (ZIP_IS_STR(entry.encoding)) {
            /* Raw compare */
            if (entry.len == slen) {
                return memcmp(p+entry.headersize,sstr,slen) == 0;
            } else {
                return 0;
            }
        } else {
            /* Try to compare encoded values. Don't compare encoding because
            * different implementations may encoded integers differently. */
            if (zipTryEncoding(sstr,slen,&sval,&sencoding)) {
            zval = zipLoadInteger(p+entry.headersize,entry.encoding);
            return zval == sval;
            }
        }
        return 0;
    }

    /* Find pointer to the entry equal to the specified entry. Skip 'skip' entries
    * between every comparison. Returns NULL when the field could not be found. */
    unsigned char *ziplistFind(unsigned char *p, unsigned char *vstr, unsigned int vlen, unsigned int skip) {
        int skipcnt = 0;
        unsigned char vencoding = 0;
        long long vll = 0;

        while (p[0] != ZIP_END) {
            unsigned int prevlensize, encoding, lensize, len;
            unsigned char *q;

            ZIP_DECODE_PREVLENSIZE(p, prevlensize);
            ZIP_DECODE_LENGTH(p + prevlensize, encoding, lensize, len);
            q = p + prevlensize + lensize;

            if (skipcnt == 0) {
                /* Compare current entry with specified entry */
                if (ZIP_IS_STR(encoding)) {
                    if (len == vlen && memcmp(q, vstr, vlen) == 0) {
                        return p;
                    }
                } else {
                    /* Find out if the searched field can be encoded. Note that
                    * we do it only the first time, once done vencoding is set
                    * to non-zero and vll is set to the integer value. */
                    if (vencoding == 0) {
                        if (!zipTryEncoding(vstr, vlen, &vll, &vencoding)) {
                            /* If the entry can't be encoded we set it to
                            * UCHAR_MAX so that we don't retry again the next
                            * time. */
                            vencoding = UCHAR_MAX;
                        }
                        /* Must be non-zero by now */
                        assert(vencoding);
                    }

                    /* Compare current entry with specified entry, do it only
                    * if vencoding != UCHAR_MAX because if there is no encoding
                    * possible for the field it can't be a valid integer. */
                    if (vencoding != UCHAR_MAX) {
                        long long ll = zipLoadInteger(q, encoding);
                        if (ll == vll) {
                            return p;
                        }
                    }
                }

                /* Reset skip count */
                skipcnt = skip;
            } else {
                /* Skip entry */
                skipcnt--;
            }

            /* Move to next entry */
            p = q + len;
        }

        return NULL;
    }

    /* Return length of ziplist. */
    unsigned int ziplistLen(unsigned char *zl) {
        unsigned int len = 0;
        if (intrev16ifbe(ZIPLIST_LENGTH(zl)) < UINT16_MAX) {
            len = intrev16ifbe(ZIPLIST_LENGTH(zl));
        } else {
            unsigned char *p = zl+ZIPLIST_HEADER_SIZE;
            while (*p != ZIP_END) {
                p += zipRawEntryLength(p);
                len++;
            }

            /* Re-store length if small enough */
            if (len < UINT16_MAX) ZIPLIST_LENGTH(zl) = intrev16ifbe(len);
        }
        return len;
    }

    /* Return ziplist blob size in bytes. */
    size_t ziplistBlobLen(unsigned char *zl) {
        return intrev32ifbe(ZIPLIST_BYTES(zl));
    }

    void ziplistRepr(unsigned char *zl) {
        unsigned char *p;
        int index = 0;
        zlentry entry;

        printf(
            "{total bytes %d} "
            "{num entries %u}\n"
            "{tail offset %u}\n",
            intrev32ifbe(ZIPLIST_BYTES(zl)),
            intrev16ifbe(ZIPLIST_LENGTH(zl)),
            intrev32ifbe(ZIPLIST_TAIL_OFFSET(zl)));
        p = ZIPLIST_ENTRY_HEAD(zl);
        while(*p != ZIP_END) {
            zipEntry(p, &entry);
            printf(
                "{\n"
                    "\taddr 0x%08lx,\n"
                    "\tindex %2d,\n"
                    "\toffset %5ld,\n"
                    "\thdr+entry len: %5u,\n"
                    "\thdr len%2u,\n"
                    "\tprevrawlen: %5u,\n"
                    "\tprevrawlensize: %2u,\n"
                    "\tpayload %5u\n",
                (long unsigned)p,
                index,
                (unsigned long) (p-zl),
                entry.headersize+entry.len,
                entry.headersize,
                entry.prevrawlen,
                entry.prevrawlensize,
                entry.len);
            printf("\tbytes: ");
            for (unsigned int i = 0; i < entry.headersize+entry.len; i++) {
                printf("%02x|",p[i]);
            }
            printf("\n");
            p += entry.headersize;
            if (ZIP_IS_STR(entry.encoding)) {
                printf("\t[str]");
                if (entry.len > 40) {
                    if (fwrite(p,40,1,stdout) == 0) perror("fwrite");
                    printf("...");
                } else {
                    if (entry.len &&
                        fwrite(p,entry.len,1,stdout) == 0) perror("fwrite");
                }
            } else {
                printf("\t[int]%lld", (long long) zipLoadInteger(p,entry.encoding));
            }
            printf("\n}\n");
            p += entry.len;
            index++;
        }
        printf("{end}\n\n");
    }

    /* Convert a string into a long long. Returns 1 if the string could be parsed
    * into a (non-overflowing) long long, 0 otherwise. The value will be set to
    * the parsed value when appropriate.
    *
    * Note that this function demands that the string strictly represents
    * a long long: no spaces or other characters before or after the string
    * representing the number are accepted, nor zeroes at the start if not
    * for the string "0" representing the zero number.
    *
    * Because of its strictness, it is safe to use this function to check if
    * you can convert a string into a long long, and obtain back the string
    * from the number without any loss in the string representation. */
    int string2ll(const char *s, size_t slen, long long *value) {
        const char *p = s;
        size_t plen = 0;
        int negative = 0;
        unsigned long long v;

        /* A zero length string is not a valid number. */
        if (plen == slen)
            return 0;

        /* Special case: first and only digit is 0. */
        if (slen == 1 && p[0] == '0') {
            if (value != NULL) *value = 0;
            return 1;
        }

        /* Handle negative numbers: just set a flag and continue like if it
        * was a positive number. Later convert into negative. */
        if (p[0] == '-') {
            negative = 1;
            p++; plen++;

            /* Abort on only a negative sign. */
            if (plen == slen)
                return 0;
        }

        /* First digit should be 1-9, otherwise the string should just be 0. */
        if (p[0] >= '1' && p[0] <= '9') {
            v = p[0]-'0';
            p++; plen++;
        } else {
            return 0;
        }

        /* Parse all the other digits, checking for overflow at every step. */
        while (plen < slen && p[0] >= '0' && p[0] <= '9') {
            这里判断溢出是用的先除和先减的方法，因为如果是先乘或者先加那么可能已经溢出了。
            if (v > (ULLONG_MAX / 10)) /* Overflow. */
                return 0;
            v *= 10;

            if (v > (ULLONG_MAX - (p[0]-'0'))) /* Overflow. */
                return 0;
            v += p[0]-'0';

            p++; plen++;
        }

        /* Return if not all bytes were used. */
        if (plen < slen)
            return 0;

        /* Convert to negative if needed, and do the final overflow check when
        * converting from unsigned long long to long long. */
        if (negative) {
            if (v > ((unsigned long long)(-(LLONG_MIN+1))+1)) /* Overflow. */
                return 0;
            if (value != NULL) *value = -v;
        } else {
            if (v > LLONG_MAX) /* Overflow. */
                return 0;
            if (value != NULL) *value = v;
        }
        return 1;
    }
