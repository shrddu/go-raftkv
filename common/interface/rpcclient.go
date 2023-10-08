package _interface

import (
	"go-raftkv/common/rpc"
	"time"
)

type RpcClientI interface {
	Send(request *rpc.MyRequest) rpc.Reply
	SendWithTimeout(*rpc.MyRequest, time.Duration) rpc.Reply
}
