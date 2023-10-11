package rpc

// 空接口 表示结果或回复
type Reply interface{}
type ClientRPCReply struct {
	Success bool
	Entry   LogEntry
}

type ServerRPCReply struct {
	/** 当前的任期号，用于领导人去更新自己 */
	Term int64

	/** 跟随者包含了匹配上 prevLogIndex 和 prevLogTerm 的日志时为真  */
	Success bool
}

// 候选人请求的请求结果
type ReqVoteResult struct {
	// 当前服务器的周期
	Term int64
	// 是否投给他一票
	IsVoted bool
}

// 服务器进行appendentries操作时的返回结果
type AppendResult struct {
	// 用于添加失败时让请求者知道并且更改他的周期
	Term    int64
	Success bool
}
type MemberAddResult struct {
	IsSuccess  bool
	LeaderAddr string
	Term       int64
}
type SyncPeersResult struct {
	IsSuccess bool
}
