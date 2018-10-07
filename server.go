package main

import (
	"net"
	"fmt"
	"time"
	"sync"
	"net/rpc/jsonrpc"
	"net/rpc"
)

var wg sync.WaitGroup

var (
	queue     []TaskUnit
	queueLock sync.Mutex
)

type TaskUnit struct {
	TaskName     string
	TaskDoneChan chan bool
}

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
