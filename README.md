# aredis
该项目的主要目标有两个:
- 以redis-6.0.8版本为例分析redis工作原理。在整理过程中可能会去掉一些和系统或者编译器版本相关的分支判断,而是以linux环境为准。
- 使用其他语言来重新实现redis作为练习。基本步骤是：  
    1. 开发请求处理模块，即网络连接和通信协议。
    2. 开发核心功能，比如redis的核心数据结构及其操作。
    3. 开发附加功能，比如主从，集群等。
    4. 有些功能需要看是否能实现，或者是否有替代方案，比如Lua脚本。
