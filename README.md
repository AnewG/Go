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
