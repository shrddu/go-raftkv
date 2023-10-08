/*
*

	@author: 18418
	@date: 2023/9/24
	@desc:  一次启动多个serverNode

*
*/
package main

import (
	"flag"
	"go-raftkv/server/node"
	"time"
)

// ldflags 标签注入port
var Port = "9990"
var PeerAddrs = []string{"localhost:9990", "localhost:9991", "localhost:9992"}

func main() {
	flag.Parse()
	cfg := &node.Config{SelfPort: Port, PeerAddrs: PeerAddrs}
	newNode := &node.Node{Config: cfg}

	newNode.Init()
	time.Sleep(time.Minute)
}
