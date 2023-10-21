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
)

/*
	程序主入口，一次启动一个，可以再build参数中添加port标签来区分server，在PeerAddrs中设置已知服务器
	在运行中添加新服务器时需要，在参数中声明IsNewNode为string类型的true 比如 : -ldflags "-X main.Port=9995 -X main.IsNewNode=true"
*/

// ldflags 标签注入port
var Port = "9990"
var IsNewNode string = "false"
var PeerAddrs = []string{"localhost:9990", "localhost:9991", "localhost:9992"}

func main() {
	flag.Parse()
	cfg := &node.Config{SelfPort: Port, PeerAddrs: PeerAddrs, IsNewNode: IsNewNode}
	newNode := &node.Node{Config: cfg}
	newNode.Init()
}
