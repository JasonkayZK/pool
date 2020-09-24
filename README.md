## Channel Pool

 [![Build Status](https://travis-ci.org/JasonkayZK/pool.svg?branch=channel-pool)](https://github.com/JasonkayZK/pool/tree/channel-pool/) ![Go](https://img.shields.io/github/go-mod/go-version/JasonkayZK/pool) ![repo-size](https://img.shields.io/github/repo-size/JasonkayZK/pool) ![stars](https://img.shields.io/github/stars/JasonkayZK/pool?style=social) ![MIT License](https://img.shields.io/github/license/JasonkayZK/pool)

利用Channel实现的高性能连接池

### 功能

-   通用连接池：适合任意需要连接池的场景；
-   自动关闭空闲连接：可设置连接的最大空闲时间，超时的连接将关闭丢弃，避免空闲时连接自动失效问题；
-   连接健康检查：支持用户设定 ping 方法，检查连接的连通性，无效的连接将丢弃；

### 基本用法

