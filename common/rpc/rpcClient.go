package rpc

import (
	"context"
	"github.com/smallnest/rpcx/client"
	log2 "go-raftkv/common/log"
	"go.uber.org/zap"
	"time"
)

type RpcClient struct {
}

var Log *zap.SugaredLogger

func (dc *RpcClient) Send(request *MyRequest) Reply {
	return dc.SendWithTimeout(request, 5*time.Second)
}
func (dc *RpcClient) SendWithTimeout(request *MyRequest, time time.Duration) Reply {
	dc.Init()
	//区分一下client请求和server请求
	switch request.RequestType {
	case CLIENT_REQ:
		d, _ := client.NewPeer2PeerDiscovery("tcp@"+request.ServiceId, "")
		option := client.DefaultOption
		option.ConnectTimeout = time
		xclient := client.NewXClient(request.ServicePath, client.Failtry, client.RandomSelect, d, option)
		defer xclient.Close()
		replay := &ClientRPCReply{}
		call, err := xclient.Go(context.Background(), request.ServiceMethod, request.Args, replay, nil)
		if err != nil {
			Log.Errorf("failed to send a ClientRPC Request, %+v", err)
		}
		replyCall := <-call.Done
		if replyCall.Error != nil {
			Log.Fatalf("failed to call : %+v", replyCall.Error)
		} else {
			Log.Infof("success to call %s,args: %+v ,reply : %+v", request.ServiceMethod, replyCall.Args, replyCall.Reply)
		}

		return replyCall.Reply
		// TODO 完善switch
	}
	return nil

}

func (dc *RpcClient) Init() {
	Log = log2.GetLog()
}

func (dc *RpcClient) Destroy() {

}
