# Go 实现任务分发

最近在写一个服务端程序，这个服务端程序包含很多任务单元，我把需求简化一下：

1. 要求能够动态新增、修改和删除这些任务单元
2. 大部分任务单元功能都是长期的定时触发任务

那么有这样的需求，第一反应是做成脚手架 CLI，比如是这样来配置：

```shell
server -add "TaskName"
server -delete "TaskName"
server -edit "TaskName"
```

但是后来一想不对啊，这样每次都会执行一次 server 程序就结束了，程序里的任务单元还在执行呢就给我结束了，就算每个任务单元都是 **阻塞** 的，我们启动 server 以后 **僵** 在那里，我们也要启动多个 server 程序，对于程序员来说多难受啊。

那么我们换一种思路，让 server 做为服务端，一直阻塞，有人给它发指令，告诉他新增、修改和删除这些任务单元，比如大家常用的 socket 和 http。

不过 socket 和 http 的解决方案对于我这个需求来说，实在是太重量级了，假如说做成分布式任务分发配置的时候一般我们会采用基于 http 的 Restful 解决方案来做，那么今天要用的一种实现方式就是 RPC。

## RPC

go 原生就支持 rpc，提供了通过网络访问一个对象的方法的能力。服务器需要注册对象， 通过对象的类型名暴露这个服务。注册后这个对象的输出方法就可以远程调用，这个库封装了底层传输的细节，包括序列化。

有了这样便利的方式那还等什么呢？我们把新增、修改和删除这三种功能封装好暴露出去给客户端调用就可以实现我们第一个需求啦。考虑到后期这个 CLI 可能不用 go 来写，也可能用 java 甚至其他语言，那么我们尽量用比较规范的一种 RPC 通信数据格式，比如 json，正好 go 原生的 `/net/rpc` 库里就有 jsonrpc，简直不要太方便。

ok，话不多说，动手实践一个 Demo。

> 这里有一本书：https://legacy.gitbook.com/book/smallnest/go-rpc-programming-guide/details 可以用作参考

### RPC 服务端

按照定义，首先我们要定义一个结构体，既然是任务单元，那我们就叫 TaskUnit 吧。

```go
type TaskUnit struct {
	
} 
```

然后要把这个对象里的方法暴露出去，rpc 暴露出去的函数有规范：

- 方法的类型是可输出的 (the method's type is exported)
- 方法本身也是可输出的 （the method is exported）
- 方法必须由两个参数，必须是输出类型或者是内建类型 (the method has two arguments, both exported or builtin types)
- 方法的第二个参数是指针类型 (the method's second argument is a pointer)
- 方法返回类型为 error (the method has return type error)

既然它都说的这么详细了，那我们尽管拿来用，我们给 TaskUnit 申明两个方法 Add 和 Delete：

```go
// args 表示 rpc 调用的时候客户端传过来的参数
// resp 表示 rpc 调用结束时给客户端返回的结果
func (t *TaskUnit) Add(args string, resp *bool) error {
	fmt.Println("[RPC] ADD TASK")
	*resp = true
	return nil
}

func (t *TaskUnit) Delete(args string, resp *bool) error {
	fmt.Println("[RPC] DEL TASK")
	*resp = true
	return nil
}
```

这里因为是个 Demo 所以简化了一下，客户端在执行 rpc 调用的时候给 args 赋值成我们的任务名称，所以是 string 类型，服务端这边给客户端返回简单的一个结果，例如告诉他是否执行成功就好了，所以设置成了 bool 型的指针，为什么是指针？因为 rpc 对暴露出去函数规范里要求。

接下来我们在 main 函数里注册这个对象，设置好监听端口，等待客户端的调用：

```go
func main() {
	rpc.Register(new(TaskUnit))

	l, err := net.Listen("tcp", ":3333")
	if err != nil {
		fmt.Printf("[RPC] Listener tcp err: %s", err)
		return
	}

	for {
		fmt.Println("[RPC] wating...")
		conn, err := l.Accept()
		if err != nil {
			fmt.Sprintf("[RPC] accept connection err: %s\n", conn)
		}
		go jsonrpc.ServeConn(conn)
	}
}
```

### RPC 客户端

做为 rpc 客户端实现起来就简单多了，只要连接上服务端，调用 Call 方法调用服务端暴露出来的函数，比如我们刚才暴露出来的 TaskUnit 对象的 Add 函数，并传递相关参数，获取返回结果即可。

```go
func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:3333", 1000*1000*1000*30)
	if err != nil {
		fmt.Printf("create client err:%s\n", err)
		return
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)
	
    // 暴露出来的函数是 TaskUnit 对象里的 Add 方法
    // 传递的是 TaskName
    // 返回的是 reply
    var reply bool
	err = client.Call("TaskUnit.Add", "TaskName", &reply)

	fmt.Printf("reply: %s, err: %s\n", reply, err)
}
```

但这种方式显然不够完美，我要把它改成 CLI 来操作，比如：

```shell
client add "TaskName"
client del "TaskName"
```

这里我们会用到一个包 `github.com/urfave/cli` 来改造一下这个客户端，把 rpc 过程调用函数封装一下，最后的代码长成这个样子：

```go
// Func: Remote Procedure Call
func rpc(method string, task string)  {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:3333", 1000*1000*1000*30)
	if err != nil {
		fmt.Printf("create client err:%s\n", err)
		return
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)

	var reply bool
	err = client.Call(method, task, &reply)

	fmt.Printf("reply: %s, err: %s\n", reply, err)
}

func main() {
	app := cli.NewApp()
	app.Name = "TaskAgent"
	app.Usage = "To Add or Del Task in Server"
	app.Commands = []cli.Command{
		{
			Name:"add",
			Aliases: []string{"add"},
			Usage:   "Add Task",
			Action:  func(c *cli.Context) error {
				taskName := c.Args().First()
				rpc("TaskUnit.Add", taskName)
				return nil
			},
		},
		{
			Name:"del",
			Aliases: []string{"del"},
			Usage:   "Delete Task",
			Action:  func(c *cli.Context) error {
				taskName := c.Args().First()
				rpc("TaskUnit.Delete", taskName)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
```

ok，客户端这边基本没有什么问题了，接下来让我们完善服务端那边任务创建和删除功能吧。

## 任务添加与删除

之前我们定义的 TaskUnit 结构体只实现了两个暴露的方法，结构体没有任何变量，实际上一个任务单元应该至少有一个任务名称，加上我们有一个需求是这个任务单元能够停止，而且这个任务单元是长期在后台执行的一个协程，因此我们每个任务单元应该还包含一个管道，用来接收停止信号。

所以我们首先要把 TaskUnit 结构体稍微改一下：

```go
type TaskUnit struct {
	TaskName     string
	TaskDoneChan chan bool
}
```

### 添加任务

为了达到能够后台执行一个协程不阻塞，我也是绞尽脑汁，最终翻到 sync 库中的 `sync.WaitGroup` 用法：它能够一直等到所有的 goroutine 执行完成，并且阻塞主线程的执行，直到所有的 goroutine 执行完成。

那么我们可以申明一个全局的 waitgroup ，在添加一个任务的时候，往这个 waitgroup 添加一个协程；相同的，在协程收到停止信号时，从 waitgroup 中把它移除掉。

```go
var wg sync.WaitGroup
```

还有就是为了让我们更好地在添加和删除任务时候判断一下是否有相同名称的任务，亦或者删除时任务不存在的情况，我们申明一个全局的任务列表 queue，有任务加入时给列表添加元素 ，删除任务的时候移除元素，当然考虑到后面可能有多个 CLI 同时在操作，我们最好是加个锁。

```go
var (
	queue     []TaskUnit
	queueLock sync.Mutex
)
```

ok，我们来实现一下任务添加功能，我们这里简化一下，任务就是每秒把任务名称打印一下，收到任务结束信号时，结束计时器并从 waitgroup 中删除。

```go
// Func: Execute Task
func execute(t TaskUnit) {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case c := <-ticker.C:
			fmt.Println("Task "+t.TaskName+" Tick at", c)
		case <-t.TaskDoneChan:
			fmt.Println("Task " + t.TaskName + " Done")
			ticker.Stop()
			wg.Done()
			return
		}
	}
}

// Func: Add Task
func (t *TaskUnit) Add(args string, resp *bool) error {
	fmt.Println("[RPC] ADD TASK " + args)

	t.TaskName = args
	t.TaskDoneChan = make(chan bool, 1)

	queueLock.Lock()
	queue = append(queue, *t)
	queueLock.Unlock()

	// Add A Task Goroutine
	wg.Add(1)
	go execute(*t)

	*resp = true
	return nil
}
```

### 删除任务

好了，添加任务已经实现了，接下来我们实现一下删除任务，那么删除任务的逻辑就是，从任务队列中找到相同任务名称的对象，关闭它的管道，并从列表中移除。

```go
// Func: Delete Task
func (t *TaskUnit) Delete(args string, resp *bool) error {
	fmt.Println("[RPC] DEL TASK " + args)

	found := false
	for i := 0; i < len(queue); i++ {
		fmt.Println(queue[i].TaskName)
		if queue[i].TaskName == args {
			close(queue[i].TaskDoneChan)
			queueLock.Lock()
			queue = append(queue[:i], queue[i+1:]...)
			queueLock.Unlock()
			found = true
			break
		}
	}

	if !found {
		fmt.Println("[RPC] NOT FOUND TASK " + args)
	}

	*resp = true
	return nil
}
```

## 运行一下

我们把编译好的 server 和 client 在两个 terminal 里打开。

- 在客户端这边添加两个任务再删除一个任务

![Screenshot from 2018-10-07 17-04-30](/home/k/Pictures/Screenshot from 2018-10-07 17-04-30.png)

- 服务端显示

![Screenshot from 2018-10-07 17-05-14](/home/k/Pictures/Screenshot from 2018-10-07 17-05-14.png)

## 小结

总结一下涉及的知识点：

- JsonRPC
- sync.waitgroup
- 管道

因为是 Demo 所以目前没有实现配置功能，有兴趣的小伙伴可以实现一下。

代码地址：https://github.com/MrNullPoint/GoTask.git