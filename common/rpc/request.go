package rpc

var (
	// R_VOTE 请求投票
	R_VOTE = 0
	// A_ENTRIES server之间附加日志
	A_ENTRIES = 1
	// CLIENT_REQ 客户端请求客户端 get / put
	CLIENT_REQ = 2
	// CHANGE_CONFIG_ADD 配置变更. add
	CHANGE_CONFIG_ADD = 3
	// CHANGE_CONFIG_REMOVE 配置变更. remove
	CHANGE_CONFIG_REMOVE = 4

	CLIENT_REQ_TYPE_SET = 10
	CLIENT_REQ_TYPE_GET = 11
)

// 参数接口，用于给Request的Args属性赋值
type Args interface{}

// client调用server或者server之间相互调用时的参数
type MyRequest struct {
	RequestType   int    // 请求类型，用于server端判断该用handler
	ServiceId     string // ip:port
	ServicePath   string // 服务路径，一般和服务注册的名称相同
	ServiceMethod string // 所调用服务的方法 ，该service端方法需要首字母大写
	Args          Args   // request 参数 一般需要指定专门的类型
}

// 客户端请求server进行kv操作时的请求
type ClientRPCArgs struct {
	Tp int // put / set
	K  string
	V  string
}

// 日志信息
type LogEntry struct {
	Index int64  //当前server的下一个index
	Term  int64  // 当前server周期
	K     string // key
	V     string // value
}

// 服务器进行appendentries操作时的请求参数
type AppendEntriesArgs struct {
	/** 发送者的任期号  */
	Term int64

	/** 被请求者 ID(ip:selfPort) */
	ServerId string

	/** 领导人的 Id，以便于跟随者重定向请求 */
	LeaderId string

	/**新的日志条目紧随之前的索引值  */
	PrevLogIndex int64

	/** prevLogIndex 条目的任期号  */
	PreLogTerm int64

	/** 准备存储的日志条目（表示心跳时为空；一次性发送多个是为了提高效率） */
	Entries *[]LogEntry

	/** 领导人已经提交的日志的索引值  */
	LeaderCommit int64
}

// 候选人进行投票请求时的参数
type ReqVoteParam struct {
	// 请求人（候选人）想要从任的任期号
	Term int64
	// 请求人的最后一次日志的index
	LastLogIndex int64
	// 请求人的最后一次日志的term
	LastLogTerm int64
	// 请求人的地址 ip:port
	ServiceId string
}
