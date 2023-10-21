package clientMethod

import (
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
)

var Addrs = []string{"localhost:9990", "localhost:9991", "localhost:9992"}

var client = rpc.RpcClient{}
var Log = log.GetLog()

func SendASetRequest(key string, value string) bool {
	for _, addr := range Addrs {
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
		rpcClient := &rpc.RpcClient{}
		reply := rpcClient.Send(request)
		if reply != nil {
			rpcReply := reply.(*rpc.ClientRPCReply)
			if rpcReply.Success {
				Log.Debug("SendSetRequest() : rpcReply.Success is true")
				break
			} else {
				Log.Debug("SendSetRequest() : rpcReply.Success is false")
				return false
			}
		}

	}
	return true
}
