package main

import (
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"strconv"
	"time"
)

var addrs = []string{"localhost:9990", "localhost:9991", "localhost:9992"}

var client = rpc.RpcClient{}
var Log = rpc.Log

func main() {
	Log = log.GetLog()

	for true {
		for i := 0; i < 10; i++ {
			// 调用10次set rpc
			success := sendSetRequest("key"+strconv.Itoa(i), "value"+strconv.Itoa(i))
			Log.Infof(":第%d次调用set命令结果%t", i+1, success)
		}
		time.Sleep(3 * time.Second)
	}

}

func sendSetRequest(key string, value string) bool {

	for _, addr := range addrs {
		request := &rpc.MyRequest{
			RequestType:   rpc.CLIENT_REQ,
			ServiceId:     addr,
			ServicePath:   "Node",
			ServiceMethod: "HandlerClientRequest",
			Args: &rpc.ClientRPCArgs{
				Tp: rpc.CLIENT_REQ_TYPE_SET,
				K:  key,
				V:  value,
			},
		}
		reply := client.Send(request)
		rpcReply := reply.(*rpc.ClientRPCReply)
		if rpcReply.Success {
			Log.Infof("sendSetRequest() : rpcReply.Success is true")
			// TODO 做出反馈，可以在通过cmd set或者get时返回reply的结果
			break
		} else {
			Log.Infof("sendSetRequest() : rpcReply.Success is false")
			return false
		}
	}
	return true
}
