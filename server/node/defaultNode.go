package node

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"github.com/rosedblabs/rosedb/v2"
	"github.com/smallnest/rpcx/server"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"go-raftkv/server/logmodule"
	"go-raftkv/server/membershipchange"
	"go-raftkv/server/statemachine"
	"math"
	"sync/atomic"
	"time"
)

var (
	LEADER    = 0
	CANDIDATE = 1
	FOLLOWER  = 2
)

/**
(1)结构体中部分字段为什么用指针类型而不使用值类型?
   1. 使用指针传递结构体原因是为了避免拷贝大量的数据，当将一个结构体作为值类型传递时，会发生一次完成的拷贝操作，包括结构体中的每个字段，如果结构体较大或者包含大量数据，会导致显著的性能损耗。
    相比之下，使用指针传递结构体只需要传递指向结构体的地址，而不会进行真正的拷贝操作，做到节省时间和内存，效率更高
   2. 同时使用指针还可以实现结构体字段的可选性，通过使用指针类型的字段，可以将其设为nil来表示起字段不需要使用或者没有提供值，在部分场景下非常有用。
    使用指针传递结构体可以提高性能并允许字段的可选性。然而，如果结构体较小或者包含少量字段，使用值类型传递也是可以的，因为拷贝的开销相对较小。选择指针传递还是值类型传递取决于具体的情况和需求。
(2)当父协程是main协程时，父协程退出，父协程下的所有子协程也会跟着退出；当父协程不是main协程时，父协程退出，父协程下的所有子协程并不会跟着退出（子协程直到自己的所有逻辑执行完或者是main协程结束才结束）
*/

//	Node server节点
//	@Description: 注意调用init方法
//
// @Note : 属性中时间的单位都是毫秒
type Node struct {
	// 配置selfport 和 peerAddrs
	Config *Config

	// 选举时间超时限制
	ElectionTime int64
	// 上一次选举时间 可以通过append方法触发或者RequestVote 单位milliseconds
	PreElectionTime int64
	// 选举间隔 单位milliseconds
	ElectionTick int64

	/** 上次一心跳时间戳  心跳通过append方法触发，分为 EmptyEntriesAppend和 正常的AppendEntries 单位milliseconds*/
	PreHeartBeatTime int64
	/** 心跳间隔基数 单位milliseconds*/
	HeartBeatTick int64

	/**
	节点当前状态
	*/
	// 本机角色
	State      int
	LeaderAddr string // init中未设置
	SelfAddr   string
	// 运行状态
	IsRunning bool

	// 当前周期
	CurrentTerm int64 `default:"0"`

	/*在当前获得选票的候选人的Addr*/
	VotedFor string `default:""`

	/* ============ 所有服务器上经常变的 ============= */

	/** 已知的最大的已经被提交的LogModule日志条目的索引值 */
	CommitIndex int64

	/** 最后被应用到状态机的日志条目索引值（初始化为 0，持续递增) */
	LastApplied int64

	/* ========== 在领导人里经常改变的(选举后重新初始化) ================== */

	/** 对于每一个服务器，需要发送给他的下一个日志条目的索引值（初始化为领导人最后索引值加一） 不仅初始化时要设置，当follower变成leader时也需要重新设置*/
	nextIndexs *map[string]int64 // map[serverAddr]nextIndex

	/** 对于每一个服务器，已经复制给他的日志的最高索引值 */
	copiedIndexs *map[string]int64 // map[serverAddr]nextIndex
	// 状态机
	StateMachine *statemachine.StateMachine
	/* 一致性模块 */
	Consensus *Consensus
	/* 成员变更模块 */
	Membership *membershipchange.MembershipChanges
}

var Log = log.GetLog()

type Config struct {
	SelfPort  string
	PeerAddrs []string
	// rpc客户端
	RpcClient *rpc.RpcClient
	LogModule *logmodule.LogModule
}

func (node *Node) Init() {
	Log.Infof("node start init ...")
	node.ElectionTime = 15 * 1000
	node.PreElectionTime = 0
	node.PreHeartBeatTime = 0
	node.HeartBeatTick = 5 * 100
	node.ElectionTick = 5 * 100
	node.State = FOLLOWER
	node.SelfAddr = "localhost:" + node.Config.SelfPort
	node.IsRunning = true
	// 初始化剩余的config内容
	node.Config.RpcClient = &rpc.RpcClient{}
	node.Config.LogModule = node.getLogModuleInstance()
	node.nextIndexs = &map[string]int64{}
	node.copiedIndexs = &map[string]int64{}
	node.StateMachine = statemachine.GetInstance(node.Config.SelfPort)
	node.Consensus = NewConsensus(node)
	// TODO 完善membershipchange
	node.Membership = &membershipchange.MembershipChanges{}

	//TODO LinkedBlockingQueue 一个队列
	//处理LinkedBlockingQueue中存储的的失败事件
	//TODO ReplicationFailQueueConsumer 一个处理队列内失败任务的消费者
	if lastEntry := node.Config.LogModule.GetLastEntry(); lastEntry != nil {
		node.CurrentTerm = lastEntry.Term
	}
	// 异步开启
	go func() {
		node.initTickerWork()
	}()

	Log.Infof("%s start success ...", node.SelfAddr)
	// 所有配置初始化完成后注册server
	s := server.NewServer()
	err := s.Register(node, "")
	if err != nil {
		panic(err)
	}
	err = s.Serve("tcp", "localhost:"+node.Config.SelfPort)
	if err != nil {
		panic(err)
	}
}
func (node *Node) initTickerWork() {
	// 启动心跳任务
	go func(t *time.Ticker) {
		defer t.Stop()
		for range t.C {
			node.heartBeatTask()
		}
	}(time.NewTicker(time.Duration(node.HeartBeatTick) * time.Millisecond))

	// 启动选举任务
	time.Sleep(3000 * time.Millisecond)
	go func(t *time.Ticker) {
		defer t.Stop()
		for range t.C {
			node.electionTask()
		}
	}(time.NewTicker(time.Duration(node.ElectionTick) * time.Millisecond))

}
func (node *Node) getLogModuleInstance() *logmodule.LogModule {
	dirPath := statemachine.BasePath + node.Config.SelfPort + "/logmodule/"
	options := rosedb.DefaultOptions
	options.DirPath = dirPath
	db, err := rosedb.Open(options)
	if err != nil {
		Log.Panic(err)
	}
	module := &logmodule.LogModule{
		DBDir:   statemachine.BasePath + node.Config.SelfPort + "/statemachine/",
		LogsDir: dirPath,
		DB:      db,
	}
	module.Init()
	Log.Infof("get a logmodule : %+v", module)
	return module
}

// heartBeatTask
//
//	@Description: 只有leader才可以发送心跳，follower接受心跳并更新term和commitIndex
//	@receiver node
func (node *Node) heartBeatTask() {
	if node.State != LEADER {
		return
	}
	if time.Now().UnixMilli()-node.PreHeartBeatTime < node.HeartBeatTick {
		return
	}
	Log.Infoln(node.SelfAddr + " start a heartBeat")
	peers := node.getPeerAddrsWithOutSelf()
	for _, peer := range peers {
		args := &rpc.AppendEntriesArgs{
			Term:         node.CurrentTerm,
			ServerId:     peer,
			LeaderId:     node.SelfAddr,
			Entries:      nil,
			LeaderCommit: node.CommitIndex,
		}
		request := &rpc.MyRequest{
			RequestType:   rpc.A_ENTRIES,
			ServiceId:     peer,
			ServicePath:   "Node",
			ServiceMethod: "HandlerAppendEntries",
			Args:          args,
		}
		/*异步发送心跳*/
		peerCopy := peer
		go func() {
			reply := node.Config.RpcClient.Send(request)
			if reply != nil {
				result := reply.(*rpc.AppendResult)
				term := result.Term
				if term > node.CurrentTerm {
					Log.Warnf("node %s will be a follower,self term : %d ,new term : %d ", node.SelfAddr, node.CurrentTerm, term)
					node.CurrentTerm = term
					node.VotedFor = ""
					node.State = FOLLOWER
				}
			} else {
				Log.Warnf("node %s send a heartbeat to %s failed ", node.SelfAddr, peerCopy)
			}
		}()
	}

}
func (node *Node) getPeerAddrsWithOutSelf() []string {
	peers := node.Config.PeerAddrs
	peersWithoutSelf := make([]string, len(peers)-1)
	index := 0
	for _, peer := range peers {
		if peer != node.SelfAddr {
			peersWithoutSelf[index] = peer
			index++
		}
	}
	return peersWithoutSelf
}

// electionTask
//
//	@Description: 选举任务，只有follower才能发起选举任务从而获取选票。
//	@receiver node
func (node *Node) electionTask() {
	// 1. 若上次选举过期，将自身变为候选者开始选举
	if node.State == LEADER {
		return
	}
	Log.Infof("now time is %d,subtraction preElectionTime result is : %d", time.Now().UnixMilli(), time.Now().UnixMilli()-node.PreElectionTime)
	if time.Now().UnixMilli()-node.PreElectionTime < node.ElectionTick || time.Now().UnixMilli()-node.PreHeartBeatTime < 3*node.HeartBeatTick {
		return
	}
	node.State = CANDIDATE
	Log.Infof("node %s start election,current term %d", node.SelfAddr, node.CurrentTerm)
	node.PreElectionTime = time.Now().UnixMilli()
	// 2. 向其他服务器发送vote请求
	node.VotedFor = node.SelfAddr
	peers := node.getPeerAddrsWithOutSelf()
	Log.Infof("node %s will send vote request to peers contains %+v", node.SelfAddr, peers)
	lastLogIndex := node.Config.LogModule.GetLastIndex()
	param := &rpc.ReqVoteParam{
		Term:         node.CurrentTerm,
		LastLogIndex: lastLogIndex,
		ServiceId:    node.SelfAddr,
	}
	if lastLogIndex == 0 && node.CurrentTerm == 0 {
		param.Term = 0
	} else {
		param.Term = node.Config.LogModule.Get(lastLogIndex).Term
	}
	c := make(chan bool, len(peers))
	var isChannelClosed int64 = 0
	for _, peer := range peers {
		peer := peer
		go func() {
			request := &rpc.MyRequest{
				RequestType:   rpc.R_VOTE,
				ServiceId:     peer,
				ServicePath:   "Node",
				ServiceMethod: "HandlerRequestVote",
				Args:          param,
			}
			result := node.Config.RpcClient.Send(request)
			if result != nil {
				voteResult := result.(*rpc.ReqVoteResult)
				if voteResult.IsVoted {
					if atomic.LoadInt64(&isChannelClosed) == 0 {
						c <- true
					}
				} else {
					peerTerm := voteResult.Term
					if peerTerm > node.CurrentTerm {
						node.CurrentTerm = peerTerm
					}
					return
				}
			} else {
				Log.Warnf("ReqVote from node %s to %s failed , reply is nil", node.SelfAddr, peer)
				if atomic.LoadInt64(&isChannelClosed) == 0 {
					c <- false
				}

			}

		}()
	}
	var votedCount int64 = 0
	var sendedCount int64 = 0
	// 3. 监听结果，如果半数以上同意则自己term自增1，设置自己为leader
	for msg := range c {
		switch msg {
		case true:
			// 如果监听过程中被发送心跳导致状态改变，那就直接停止选举
			if node.State == FOLLOWER {
				return
			}
			atomic.AddInt64(&votedCount, 1)
			atomic.AddInt64(&sendedCount, 1)
			if votedCount >= (int64)(len(peers)/2) {
				// 公投成功
				node.State = LEADER
				node.LeaderAddr = node.SelfAddr
				node.CurrentTerm += 1
				node.VotedFor = ""
				Log.Infof("node %s vote success,new term is %d , now will call 'becomeLeaderToDo()'method ", node.SelfAddr, node.CurrentTerm)
				node.becomeLeaderToDo()
				break
			}
			break
		case false:
			{
				Log.Warnf("electionTask : listening a 'false' msg from channel")
				atomic.AddInt64(&sendedCount, 1)
				break
			}
		}
		if sendedCount == int64(len(peers)) || node.State == LEADER {
			atomic.AddInt64(&isChannelClosed, 1)
			break
		}
	}

	node.PreElectionTime = time.Now().UnixMilli()
}

/**
 * 初始化所有的 nextIndex 值为自己的最后一条日志的 index + 1. 如果下次 RPC 时, 跟随者和leader 不一致,就会失败.
 * 那么 leader 尝试递减 nextIndex 并进行重试.最终将达成一致.
 */
//
// becomeLeaderToDo
//  @Description: 更新nextIndexs和commitIndex，创建空日志并同步到其他节点，从而自动同步之前的日志
//  @receiver node
//
func (node *Node) becomeLeaderToDo() {
	peers := node.getPeerAddrsWithOutSelf()
	// 该空entry的index为0，则每个server的logModule的第一个log起表示当前term的作用
	termEntry := &rpc.LogEntry{
		Index: 0,
		Term:  node.CurrentTerm,
		K:     "",
		V:     "",
	}
	// 提交到本地logModule
	node.Config.LogModule.Set(termEntry)
	var count int64 = 0
	var receivedCount int64 = 0
	var channelIsClosed int64 = 0
	c := make(chan bool, len(peers))
	for _, peer := range peers {
		peer := peer
		go func() {
			isSuccess := node.replication(peer, termEntry)
			if isSuccess {
				if atomic.LoadInt64(&channelIsClosed) == 0 {
					c <- true
				}
			} else {
				if atomic.LoadInt64(&channelIsClosed) == 0 {
					c <- false
					Log.Warnf("replication form %s to %s has failed", node.SelfAddr, peer)
				}
			}
		}()
	}
	for msg := range c {
		atomic.AddInt64(&receivedCount, 1)
		if msg {
			atomic.AddInt64(&count, 1)
			if count >= (int64)(len(peers)/2) {
				if channelIsClosed != 0 {
					break
				}
				node.StateMachine.Set(termEntry)
				Log.Infof("success to update peers' new term : %d", node.CurrentTerm)
				atomic.AddInt64(&channelIsClosed, 1)
				close(c)
			}
		}
		if receivedCount == int64(len(peers)) {
			Log.Warnf("fail to update peers' new term , leader : %s ,term : %s", node.SelfAddr, node.CurrentTerm)
			// 没有更新到所有follower，成为leader的行为失败，当前term失效，设置自己为candidate
			node.State = FOLLOWER
			atomic.AddInt64(&channelIsClosed, 1)
			close(c)
		}
	}
}

func (node *Node) replication(peer string, entry *rpc.LogEntry) bool {
	// 1. 10s内可以重复尝试(rpc超时时间3s)
	start := time.Now().UnixMilli()
	end := time.Now().UnixMilli()
	for end-start < 10*1000 {
		args := &rpc.AppendEntriesArgs{
			Term:         node.CurrentTerm,
			ServerId:     peer,
			LeaderId:     node.LeaderAddr,
			PrevLogIndex: int64(math.Max(float64(0), float64(entry.Index-1))),
			PreLogTerm:   node.Config.LogModule.Get(int64(math.Max(float64(0), float64(entry.Index-1)))).Term,
			Entries:      nil,
			LeaderCommit: node.CommitIndex,
		}
		nextIndex := (*node.nextIndexs)[peer]
		var entries []rpc.LogEntry
		// 2. 把entry前，nextIndex后的entries都发送过去
		if entry.Index > nextIndex {
			entries = make([]rpc.LogEntry, entry.Index-nextIndex)
			var j int = 0
			for i := nextIndex; i <= entry.Index; i++ {
				if value := node.Config.LogModule.Get(i); value != nil {
					entries[j] = *value
					j++
				}
			}
		} else {
			entries = make([]rpc.LogEntry, 1)
			entries[0] = *entry
		}
		// entries前面的那一个
		preLog := node.getPreLog(entries[0])
		args.PrevLogIndex = preLog.Index
		args.PreLogTerm = preLog.Term
		args.Entries = &entries
		request := &rpc.MyRequest{
			RequestType:   rpc.A_ENTRIES,
			ServiceId:     peer,
			ServicePath:   "Node",
			ServiceMethod: "HandlerAppendEntries",
			Args:          args,
		}
		reply := node.Config.RpcClient.SendWithTimeout(request, 3*1000*time.Millisecond)
		if reply != nil {
			appendResult := reply.(*rpc.AppendResult)
			if appendResult.Success {
				Log.Infof("replication logEntry {%+v}from %s to %s now is success", entry, node.SelfAddr, peer)
				(*node.nextIndexs)[peer] = entry.Index + 1
				(*node.copiedIndexs)[peer] = entry.Index
				return true
			} else {
				// 分析原因
				// 1. 对方周期大于自己
				if appendResult.Term > node.CurrentTerm {
					Log.Warnf("follower %s term is bigger than leader %s ,leader term changed to %d", peer, node.SelfAddr, appendResult.Term)
					node.CurrentTerm = appendResult.Term
					node.State = FOLLOWER
					return false
				} else {
					// 2. preLog存在但index相同，term不同，需要减小后重试
					// 3. preLog不存在，需要减小重试
					if nextIndex == 0 {
						nextIndex = 1
					}
					(*node.nextIndexs)[peer] = nextIndex - 1
					// 本次循环结束，在超时范围内自动重试
				}

			}
		} else {
			// 重试
		}
		end = time.Now().UnixMilli()
	}
	// 超时
	return false
}

func (node *Node) getPreLog(entry rpc.LogEntry) *rpc.LogEntry {
	preLog := node.Config.LogModule.Get(entry.Index - 1)
	if preLog == nil {
		Log.Warnf("get a preLog before %+v is empty", entry)
		preLog = &rpc.LogEntry{
			Index: 0,
			Term:  0,
			K:     "",
			V:     "",
		}
	}
	return preLog
}
func (node *Node) Destroy() {
	node.Config.RpcClient.Destroy()
}

// HandlerClientRequest
//
//	@Description: 指针是一个类型，类似于int、float的基本类型，而指针指向地址，指针从概念上不等同于地址，但当log输出指针时，会自动输出指针指向的地址。
//	mapstructure.Decode的第一个参数是将指针这个类型转换为地址，这个地址相当于一个interface，因为interface本质上可以理解为地址.
//
// https://www.cnblogs.com/apocelipes/p/13796041.html
// 其实也没什么好总结的。只有两点需要记住，一是interface是有自己对应的实体数据结构的，二是尽量不要用指针去指向interface，因为golang对指针自动解引用的处理会带来陷阱。
//
//	@receiver t
//	@param ctx
//	@param args 必须是明确的指针，interface{}也不行。一般来说没有必要让一个指针指向接口，但是rpcx规定args必须是指针，只好先用指针接受一个args
//	@param reply 必须是明确的指针
//	@return error
func (node *Node) HandlerClientRequest(ctx context.Context, args *any, reply *any) error {
	Log.Infof("Get a ClientRequest ,args:%+v", *args) //将指针指向的interface 取反得到interface类型，这样就可以正常通过%v来输出他的值了
	//rpcArgs := (args).(rpc.ClientRPCArgs) //将指针指向的interface自动转换为指定类型时，会掉进golang自动转换的陷阱，不要用这种方法来转换
	/*转换any为具体参数、回复类型*/
	argsStruct := &rpc.ClientRPCArgs{}
	err := mapstructure.Decode(*args, argsStruct) //mapstructure一般用来处理map[string]interface{}转换为json或者结构体的问题
	if err != nil {
		Log.Errorf("args decode fail: %+v", err)
	}
	replyStruct := &rpc.ClientRPCReply{}
	err = mapstructure.Decode(*reply, replyStruct)
	if err != nil {
		Log.Errorf("reply decode fail: %+v", err)
	}
	defer func() {
		*args = argsStruct
		*reply = replyStruct
	}()

	/*进行业务操作*/
	Log.Infof("HandlerClientRequest Method get a client request : %+v ", argsStruct)
	if node.State != LEADER {
		Log.Warnf("node %s get a clientRequest,but not a leader ,redirect to leader %s", node.SelfAddr, node.LeaderAddr)
		return node.Redirect(ctx, args, reply)
	}
	// GET
	if argsStruct.Tp == rpc.CLIENT_REQ_TYPE_GET {
		entry := node.StateMachine.Get(argsStruct.K)
		if entry != nil {
			replyStruct.Success = true
			replyStruct.Entry = *entry
			return nil
		} else {
			replyStruct.Success = false
			return nil
		}
	}
	// SET
	logEntry := &rpc.LogEntry{
		Index: node.Config.LogModule.GetLastIndex() + 1,
		Term:  node.CurrentTerm,
		K:     argsStruct.K,
		V:     argsStruct.V,
	}
	// 预提交，如果成功大于一半，就应用到状态机，否则删除该log之后的所有内容，从而回滚
	node.Config.LogModule.Set(logEntry)
	Log.Infof("PRE-COMMIT: write a entry{%+v} into node %s's logModule", logEntry, node.SelfAddr)
	// 复制到其他节点，观察成功数占比
	peers := node.getPeerAddrsWithOutSelf()
	var count int64 = 0
	var sendedCount int64 = 0
	var isChannelClosed int64 = 0
	c := make(chan bool, len(peers))
	defer close(c)
	for _, peer := range peers {
		peer := peer
		go func() {
			isSuccess := node.replication(peer, logEntry)
			if isSuccess {
				if atomic.LoadInt64(&isChannelClosed) == 0 {
					c <- true
				}
			} else {
				if atomic.LoadInt64(&isChannelClosed) == 0 {
					c <- false
				}
			}
		}()
	}
	for msg := range c {
		atomic.AddInt64(&sendedCount, 1)
		if msg {
			atomic.AddInt64(&count, 1)
			if count >= (int64)(len(peers)/2) {
				// 复制成功，提交log到stateMachine，并更新相关信息
				node.CommitIndex = logEntry.Index
				node.StateMachine.Set(logEntry)
				node.LastApplied = node.CommitIndex
				Log.Infof("success apply stateMachine, logEntry info : {%+v}", logEntry)
				replyStruct.Success = true
				replyStruct.Entry = *logEntry
				atomic.AddInt64(&isChannelClosed, 1)
				return nil
			}
		} else {
			if sendedCount == int64(len(peers)) {
				// 一半以上没接收到，需要回滚
				Log.Warnf("node %s fail apply statMachine entry:{%+v}", node.SelfAddr, logEntry)
				err := node.Config.LogModule.DeleteOnStartIndex(logEntry.Index)
				replyStruct.Success = false
				replyStruct.Entry = *logEntry
				return err
			}
		}
	}
	replyStruct.Success = false
	return nil
}

func (node *Node) HandlerRequestVote(ctx context.Context, args *any, reply *any) error {
	return node.Consensus.HandlerRequestVote(args, reply)
}
func (node *Node) HandlerAppendEntries(ctx context.Context, args *any, reply *any) error {
	return node.Consensus.HandlerAppendEntries(args, reply)
}
func (node *Node) Redirect(ctx context.Context, args *any, reply *any) error {
	request := &rpc.MyRequest{
		RequestType:   rpc.CLIENT_REQ,
		ServiceId:     node.LeaderAddr,
		ServicePath:   "Node",
		ServiceMethod: "HandlerClientRequest",
		Args:          args,
	}
	rpcReply := node.Config.RpcClient.Send(request)
	*reply = rpcReply
	return nil
}
