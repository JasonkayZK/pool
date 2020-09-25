## Channel Pool

 [![Build Status](https://travis-ci.org/JasonkayZK/pool.svg?branch=channel-pool)](https://github.com/JasonkayZK/pool/tree/channel-pool/) ![Go](https://img.shields.io/github/go-mod/go-version/JasonkayZK/pool) ![repo-size](https://img.shields.io/github/repo-size/JasonkayZK/pool) ![stars](https://img.shields.io/github/stars/JasonkayZK/pool?style=social) ![MIT License](https://img.shields.io/github/license/JasonkayZK/pool)

利用Channel实现的高性能协程池

### 功能

-   通用协程池：适合任意需要协程池的场景；
-   自动关闭空闲连接：可设置连接的最大空闲时间，超时的连接将关闭丢弃，避免空闲时连接自动失效问题；
-   请求连接超时：用户可自行设定请求连接超时时间；
-   连接健康检查：支持用户设定 ping 方法，检查连接的连通性，无效的连接将丢弃；

### 基本用法

通过下面的方式引入本仓库即可：

```bash
go get -u github.com/jasonkayzk/pool/channel_pool
```

#### 创建协程池

使用`NewChannelPool`创建一个协程池：

```go
func newChannelPool() (Pool, error) {
	ops := Options{
		InitialCap:  InitialCap,
		MaxCap:      MaximumCap,
		MaxIdle:     MaxIdleCap,
		Factory:     factory,
		Ping:        pingFunc,
		Close:       closeFunc,
		IdleTimeout: IdleTimeout,
		WaitTimeout: WaitTimeout,
	}
	return NewChannelPool(&ops)
}
```

初始化时需要传入协程池的相关参数；

各个参数说明如下：

-   InitialCap：初始化连接数；
-   MaxCap：池中最多可存在的连接数，当所有的协程连接资源都被占用时，再次获取将会被等待至多WaitTimeout时长，然后报WaitConnectionTimeoutErr错误；
-   MaxIdle：最多可闲置的协程数；
-   Factory：创建连接时使用的工厂方法；
-   Ping：健康检查时使用的Ping方法；
-   Close：关闭连接使用的方法；
-   IdleTimeout：协程的最大闲置时长(懒删除)，设置为小于等于0时不删除；
-   WaitTimeout：使用Get方法获取协程连接的最大等待时长(当协程池资源被完全使用时，使用Get获取连接会导致当前协程被阻塞！)，超过该时间会导致WaitConnectionTimeoutErr错误；

****

#### 使用连接

通过Get方法从协程池中取出连接，并通过类型转换，将interface{}转换为对应的连接类型；

然后使用连接处理逻辑，最后通过Put方法释放连接；

```go
func xxx() {
    conn, _ := p.Get()
    cli := conn.(*rpc.Client)
    defer p.Put(conn)
	
    // 使用cli连接处理逻辑
    ...
}
```

****

#### 获取池中可用连接数

通过Len方法，可以获取到线程池中可用的连接数；

例如：

```go
func TestPool_Get(t *testing.T) {
	p, err := newChannelPool()
	if err != nil {
		t.Errorf("create pool error: %s", err)
	}
	defer p.ShutDown()

	_, err = p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	// after one get, current capacity should be lowered by one.
	if p.Len() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d", InitialCap-1, p.Len())
	}
}
```

****

#### 空置超时

当协程空置超过一定时间(IdleTimeout)时，会将超时协程去除；

若IdleTimeout <= 0，则不会进行空置超时判断；

>   目前使用懒删除(Lazy)的方式进行协程空置超时判断，即：
>
>   当使用Get方法获取一个连接时，会遍历协程池，并将超时连接剔除；
>
>   以后可以使用cron定时任务的方式；

具体代码见Get方法；

****

#### 健康检查

与空置超时检测类似，对于连接的健康检查也是懒删除模式：

当用户使用Get方法时，会进行健康检查，只有通过了健康检查，才会返回连接；否则，会删除并重新创建一个连接返回！

具体代码见Get方法；

****

#### 关闭协程池

使用ShutDown方法关闭协程池；

ShutDown会将协程池的所有配置置空(防止内存泄漏、并便于垃圾回收)，同时尝试调用协程池中配置的close方法关闭各个协程；

>与Java中的线程池尽最大可能保证线程安全执行不同，这里由close方法来保证各个连接资源的关闭；
>
>这是考虑到，不同类型连接处理close的逻辑各异，同时，不少第三方连接已经提供了较为合理的关闭方式，所以就不重复造轮子了！
>
>但是还是要提醒一句：
>
><font color="#f00">**如果close处理不当，如：单个协程出现死循环无法退出，是一定会造成内存泄漏的！这里的ShutDown仅仅是尝试关闭连接，并不保证强制关闭(有需要的可以使用context改进本协程池)**</font>

<BR/>

### 性能测试

为了进行协程池的性能测试，首先我们建立一个rpc服务：

```go
type Number int

type Args struct {
	A, B int
}

func rpcServer() {
	number := new(Number)
	_ = rpc.Register(number)
	rpc.HandleHTTP()

	l, e := net.Listen("tcp", address)
	if e != nil {
		panic(e)
	}
	go http.Serve(l, nil)
}

func (n *Number) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

```

这个rpc服务注册了Number中的服务，好让我们可以远程调用其Multiply方法，返回两个数的乘积；

接下来我们分别通过协程池和非协程池(每次调用创建一个连接)进行rpc调用，并调用5000次，比较时间：

```
poolMethod elapsed:  1.6118875s
simpleMethod elapsed:  9.0750414s
```

>   具体测试代码见example目录；

经过测试，这个协程池的性能还是不错的，速度的确提升了很多；

<br/>

### 总结

本协程池参考了Github中star数较多的协程池，并根据自己的需求做了一定的修改：

-   [fatih/pool](https://github.com/fatih/pool)
-   [silenceper/pool](https://github.com/silenceper/pool)

当然，相比于Java中线程池的实现是小巫见大巫了；

关于本协程池实现的博客：

-   [Golang实现自定义协程池](https://jasonkayzk.github.io/2020/09/25/Golang实现自定义协程池/)

如果对Java中线程池实现感兴趣，并且想尝试自己写一个类似于Java这种大而全的协程池的，可以先看一下我之前写的JDK11中的线程池的源码解析；

系列文章入口：

-   [Java线程池ThreadPoolExecutor分析与实战](https://jasonkayzk.github.io/2020/02/06/Java线程池ThreadPoolExecutor分析与实战/)
-   [Java线程池ThreadPoolExecutor分析与实战续](https://jasonkayzk.github.io/2020/03/04/Java线程池ThreadPoolExecutor分析与实战续/)
-   [Java线程池ThreadPoolExecutor分析与实战终](https://jasonkayzk.github.io/2020/03/05/Java线程池ThreadPoolExecutor分析与实战终/)

