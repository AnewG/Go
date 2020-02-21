# Go

```

函数：

func[关键字] Double[函数名] (n int) (result int)[函数签名/函数类型] => 函数原型 {
	... 函数体
}

函数声明 = 函数原型 + 函数体

------------

方法：[有属主]

方法声明 = func[关键字] + 属主参数声明 + 方法原型(不含func) + 方法体

```

```

超时控制：

func main() {
	ch := make(chan string)

	go func() {
		time.Sleep(time.Second * 2)

		ch <- "result"
	}()

	select {
	case res := <-ch:
		fmt.Println(res)
	case <-time.After(time.Second * 1): // 1s后返回数据
		fmt.Println("timeout")
	}
}

所有 channel表达式(case) 都会被求值, 如果有多个 case 都可以运行，select 会随机选出一个执行。其他不会执行

如果有 default 子句，则立即执行该语句。如果没有 default 子句，select 将阻塞，直到某个通信可以运行

select 语句阻塞等待最先返回数据的 channel，当先接收到 time.After 的通道数据时，select 则会停止阻塞并执行该 case 代码。
此时就已经实现了对业务代码的超时处理

输出timeout并直接结束，fmt.Println(res) 将不会执行

```

```

任务定时：

func worker() {
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-ticker:
			// 执行定时任务
			fmt.Println("执行 1s 定时任务")
		}
	}
}

每隔 1 秒种，执行一次定时任务。

```

```

控制并发数

var limit = make(chan int, 3)

func main() {
	// …………
	for _, w := range work {
		go func() {
			limit <- 1
			w()
			<-limit
		}()
	}
	// …………
}

同时最多3个执行，因为 channel 满时 "limit <- 1" 将被阻塞

这里，limit <- 1 放在 func 内部而不是外部，原因是：
1.如果在外层，就是控制系统 goroutine 的数量，可能会阻塞 for 循环，影响业务逻辑。
2.limit 其实和逻辑无关，只是性能调优，放在内层和外层的语义不太一样。

还有一点要注意的是，如果 w() 发生 panic，那“许可证”可能就还不回去了，因此需要使用 defer 来保证。

```

```

关闭channel原则：

不要从一个 receiver 侧关闭 channel （sender将不知道已关闭仍然发送数据）

只有一个 sender 的情况，直接从 sender 端关闭

多个 sender 有 2 种情况

1.N 个 sender，一个 reciver 
    -> 新建一个通知用的 channel，要停止从 sender 接收时由 reciver(该 channel 唯一发送者) 关闭该 channel，并自己结束流程
    -> 每一个 sender 都有监听该 channel，一旦取到 ( 关闭状态下可取值，变为非阻塞立即返回 )，原先的发送 channel 停止发送并结束流程
    -> 不手动关闭原先的发送 channel，等 gc 自己回收
2.N 个 sender， M 个 receiver
    -> 沿用一个 reciver 的方案，但是由于有多个 receiver，所以需要由中间人专门处理通知用的 channel
    -> 中间人为带缓冲的 channel，sender 和 reciver 都能向中间人发送关闭通知
    -> 中间人缓冲容量若为 Num(senders+receivers)，则 sender/reciver 发送关闭通知的代码可以去掉 select+default，因为不需要处理阻塞，容量够大不会阻塞
    -> 同时 (多个) sender 和 reciver 都要监听通知用的 channel，一旦取到即结束自身流程

中间人代码
go func() {
    stoppedBy = <-toStop // 一旦接收到关闭通知
    close(stopCh)        // 关闭通知用的 channel
}()

```
