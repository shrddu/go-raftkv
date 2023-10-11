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
	d, _ := client.NewPeer2PeerDiscovery("tcp@"+request.ServiceId, "")
	option := client.DefaultOption
	option.ConnectTimeout = time
	xclient := client.NewXClient(request.ServicePath, client.Failtry, client.RandomSelect, d, option)
	defer xclient.Close()
	var replay Reply
	switch request.RequestType {
	case A_ENTRIES:
		replay = &AppendResult{}
		break
	case R_VOTE:
		replay = &ReqVoteResult{}
		break
	case CLIENT_REQ:
		replay = &ClientRPCReply{}
		break
	case MEMBERSHIP_CHANGE_ADD:
		replay = &MemberAddResult{}
		break
	case MEMBERSHIP_CHANGE_SYNC:
		replay = &SyncPeersResult{}
	}
	Log.Infof("selfNode will send a '%s' type rpc , args :%+v", request.ServiceMethod, request.Args)
	call, err := xclient.Go(context.Background(), request.ServiceMethod, request.Args, replay, nil)
	if err != nil {
		Log.Errorf("failed to send a %s type Request,error : %+v", request.ServiceMethod, err)
	}
	if call != nil {
		replyCall := <-call.Done
		if replyCall.Error != nil {
			Log.Fatalf("failed to call : %+v", replyCall.Error)
		} else {
			if replyCall.Reply != nil {
				Log.Infof("success to call %s , the result contains : {args: %+v ,reply : %+v}", request.ServiceMethod, replyCall.Args, replyCall.Reply)
				return replyCall.Reply
			} else {
				Log.Warnf("send a %s type Request successfully,but get a empty reply from %s", request.ServiceMethod, request.ServiceId)
			}
		}
	}

	return nil

}

func (dc *RpcClient) Init() {
	Log = log2.GetLog()
}

func (dc *RpcClient) Destroy() {

}
