## 扫描redis，记录prefix.txt中的key的总的数量，及这些key对应的字段的总数量。这些key都是hash。

代码如下：

        #!/bin/bash

        HOST=$1
        PORT=$2
        chmod u+x ./redis-cli

        prefix_list="`cat prefix.txt`"
        printf "%-60s %-10s %-10s\n" "keyword" "keyCount" "totalSize"
        for keyword in ${prefix_list}
        do
            if [[ "${keyword}" =~ ^[0-9a-zA-Z_]+ ]];then
                keyCount=0
                keyArr=();
                totalSize=0
                num=0
                while :
                do
                    count=0;
                    result=`./redis-cli -h ${HOST} -p ${PORT}  scan ${num} match "${keyword}*"  count 5000`
                   
                    arr=(${result// /})
                    num=${arr[0]}
                    
         
        #!/bin/bash

        HOST=$1
        PORT=$2
        chmod u+x ./redis-cli

        prefix_list="`cat prefix.txt`"
        printf "%-60s %-10s %-10s\n" "keyword" "keyCount" "totalSize"
        for keyword in ${prefix_list}
        do
            if [[ "${keyword}" =~ ^[0-9a-zA-Z_]+ ]];then
                keyCount=0
                keyArr=();
                totalSize=0
                num=0
                while :
                do
                    count=0;
                    result=`./redis-cli -h ${HOST} -p ${PORT}  scan ${num} match "${keyword}*"  count 1`
                    #和从命令行中看到的不同，这里返回的是空格分隔的值，第一个值为cursor，后面的值为各个key。
                    #转换成数组
                    arr=(${result// /})
                    #注意num并不是扫到的key的值，而是cursor，实际返回的key的数量为数组的数量减1.
                    num=${arr[0]}
                    count=$[ ${#arr[*]} - 1]

                    if [[ $count != 0 ]]
                    then 
                        for i in ${!arr[@]}
                        do
                            if [ $i == 0 ]
                            then continue;
                            else
                                #这里keyCount不能直接使用$count相加，因为我们还要用其作为key数组的索引来记录key。
                                keyCount=$[ $keyCount + 1]
                                keyArr[$keyCount]=${arr[$i]};
                            fi
                        done
                    fi

                    if [[ $num == 0 ]]
                    then
                        break;
                    fi
                done
                for var in ${keyArr[*]}
                do
                    rs=`./redis-cli -h ${HOST} -p ${PORT}  hlen ${var}`
                    totalSize=$[ $totalSize+ $rs ]

                done
                printf "%-60s %-10s %-10s\n" "$keyword" $keyCount $totalSize
            fi
        done
