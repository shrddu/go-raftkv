/*
*

	@author: 18418
	@date: 2023/10/16
	@desc:

*
*/
package server

import (
	"fmt"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"go.uber.org/zap/zapcore"
	"sync/atomic"
	"testing"
	"time"
)

var (
	Log       = log.GetLog().With(zapcore.WarnLevel)
	PeerAddrs = []string{"localhost:9990", "localhost:9991", "localhost:9992"}
	rpcClient = &rpc.RpcClient{}
	request   = &rpc.MyRequest{
		RequestType:   rpc.CLIENT_REQ,
		ServiceId:     "localhost:9990",
		ServicePath:   "Node",
		ServiceMethod: "HandlerClientRequest",
		Args: &rpc.ClientRPCArgs{
			Tp: rpc.CLIENT_REQ_TYPE_SET,
			K:  "key1",
			V:  "value1",
		},
	}
)

// 下面两种测试方法可能会因为rpcClient性能瓶颈无法真正测试server的并发
func BenchmarkSet(b *testing.B) {
	time.Sleep(3 * time.Second)
	var successCount int64 = 0
	for i := 0; i < b.N; i++ {
		reply := rpcClient.Send(request)
		rpcReply := reply.(*rpc.ClientRPCReply)
		if rpcReply.Success {
			atomic.AddInt64(&successCount, 1)
		} else {
			Log.Infof("SendSetRequest() : rpcReply.Success is false")
		}
	}
	fmt.Printf("成功完成 %d 次并发 \n", successCount)
}
func BenchmarkSetParallel(b *testing.B) {
	var successCount int64 = 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			go func() {
				reply := rpcClient.Send(request)
				if reply != nil {
					rpcReply :=
						reply.(*rpc.ClientRPCReply)
					if rpcReply.Success {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}()
		}
	})
	fmt.Printf("成功完成 %d 次并发", successCount)
}
