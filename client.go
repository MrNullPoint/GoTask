package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"net"
	"net/rpc/jsonrpc"
)

// Func: Remote Procedure Call
func call(method string, task string)  {
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
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:"add",
			Aliases: []string{"add"},
			Usage:   "Add Task",
			Action:  func(c *cli.Context) error {
				taskName := c.Args().First()
				call("TaskUnit.Add", taskName)
				return nil
			},
		},
		{
			Name:"del",
			Aliases: []string{"del"},
			Usage:   "Delete Task",
			Action:  func(c *cli.Context) error {
				taskName := c.Args().First()
				call("TaskUnit.Delete", taskName)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
