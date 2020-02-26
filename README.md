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

```

向channel发送数据

channel 里 recvq 存储那些尝试读取 channel 但被阻塞的 goroutine，sendq 则存储那些尝试写入 channel，但被阻塞的 goroutine

goroutineA, goroutineB 同时作为 receiver 阻塞等待接收

sender 发现 channel 的 recvqueue 里有 receiver 在等待着接收
就会出队一个，把 recvq 里 first 指针的推举出来，并将其加入到其维护的可运行 goroutine 队列中。
(按照 happened-before, receive 完成，send 才算 finished)

两个 receiver 在 channel 等待，这时 channel 另一边来了一个 sender 准备向 channel 发送数据
为了高效，用不着通过 channel 的 buffer 中转一次，直接从源地址把数据 copy 到目的地址就可以了
buffer 一般用于未有 receiver 时，做数据缓存

```

```

从channel接收数据

接收操作有两种写法，一种带 "ok"，反应 channel 是否关闭；一种不带 "ok"，这种写法，当接收到相应类型的零值时无法知道是真实的发送者发送过来的值，还是 channel 被关闭后，返回给接收者的默认类型的零值。

```

```

操作 channel 的情况总结

发生 panic 的情况有三种
1.向一个关闭的 channel 进行写操作
2.关闭一个 nil 的 channel
3.重复关闭一个 channel

读、写一个 nil channel 都会被永久阻塞，就算后期初始化 channel 并写入值了也会阻塞住。
nil channel: 声明但未初始化的 channel

close 逻辑比较简单，对于一个 channel，recvq 和 sendq 中分别保存了阻塞的发送者和接收者。
关闭 channel 后，对于等待接收者而言，会收到一个相应类型的零值。
对于等待发送者，会直接 panic。
所以，在不了解 channel 还有没有接收者的情况下，不能贸然关闭 channel。

```

```

读写锁

sync.RWMutex。

读之前调用 RLock() 函数，读完之后调用 RUnlock() 函数解锁
写之前调用 Lock() 函数，写完之后，调用 Unlock() 解锁。

另外，sync.Map 是线程安全的 map

```

```

值接收者和指针接收者的区别

值类型既可以调用值接收者的方法，也可以调用指针接收者的方法

指针类型既可以调用指针接收者的方法，也可以调用值接收者的方法。

也就是说，不管方法的接收者是什么类型，该类型的值和指针都可以调用，不必严格符合接收者的类型。

----------------

   ---                   值接收者                    指针接收者

值类型调用者   方法会使用调用者的一个副本，类似于“传值”   使用值的引用来调用方法，a.b() -> (&a).b()

指针类型调用者 指针被解引用为值，a.b() -> (*a).b()     实际上也是“传值”，方法里的操作会影响到调用者，类似于指针传参，拷贝了指针

----------------

func (p *Person) growUp() {  p为接收者
    p.age += 1
}

qcrao := Person{age: 18}
qcrao.growUp()               qcrao为调用者

----------------

关于自动生成

如果实现了接收者是值类型的方法，会隐含地也实现了接收者是指针类型的方法。（因为语法糖之后还是值调用）

反正则没有

因为当实现了一个接收者是指针类型的方法，如果此时自动生成一个接收者是值类型的方法
原本期望对接收者的改变（通过指针实现），现在无法实现，因为值类型会产生一个拷贝，不会真正影响调用者。

两者不一致

----------------

使用指针作为方法的接收者的理由：

方法能够修改接收者指向的值。
避免在每次调用方法时复制该值，在值的类型为大型结构体时，这样做会更加高效。

如果类型具备非原始的本质，不能被安全地复制，这种类型总是应该被共享，那就定义指针接收者的方法。

```
