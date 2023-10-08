/*
*

	@author: 18418
	@date: 2023/9/25
	@desc:

*
*/
package consensus

import (
	"github.com/mitchellh/mapstructure"
	"go-raftkv/common/log"
	"go-raftkv/common/rpc"
	"go-raftkv/server/node"
	"math"
	"strings"
	"sync"
	"time"
)

var Log = log.GetLog()

type Consensus struct {
	Node       *node.Node
	voteLock   *sync.Mutex
	appendLock *sync.Mutex
}

func NewConsensus(node *node.Node) *Consensus {
	voteLock := &sync.Mutex{}
	appendLock := &sync.Mutex{}
	return &Consensus{Node: node, voteLock: voteLock, appendLock: appendLock}
}

// HandlerRequestVote
//
//	@Description: 投票接受方的方法，如果发送方的term小于当前的term那么返回false，否则返回成功，只有对方大于等于自己才能返回成功。
//	term符合要求的情况下，如果lastLog的term小于自己的也是不行的，同样的index小于自己也是不行的
//	@receiver con
//	@param args
//	@param reply
//	@return error
func (con *Consensus) HandlerRequestVote(args *any, reply *any) error {
	reqVoteParam := &rpc.ReqVoteParam{}
	if err := mapstructure.Decode(*args, reqVoteParam); err != nil {
		return err
	}
	reqVoteResult := &rpc.ReqVoteResult{}
	if err := mapstructure.Decode(*reply, reqVoteResult); err != nil {
		return err
	}
	defer func() {
		*args = reqVoteParam
		*reply = reqVoteResult
	}()
	Log.Infof("%s get a vote req ,args : %+v ", con.Node.SelfAddr, reqVoteParam)

	// 如果获取锁失败
	if !con.voteLock.TryLock() {
		reqVoteResult.IsVoted = false
		reqVoteResult.Term = con.Node.CurrentTerm
		return nil
	}
	defer con.voteLock.Unlock()
	// 如果对方任期小于自己
	if reqVoteParam.Term < con.Node.CurrentTerm {
		reqVoteResult.IsVoted = false
		reqVoteResult.Term = con.Node.CurrentTerm
		return nil
	}
	if con.Node.VotedFor == "" || strings.EqualFold(con.Node.VotedFor, reqVoteParam.ServiceId) {
		lastEntry := con.Node.Config.LogModule.GetLastEntry()
		if lastEntry != nil {
			// term符合的情况下，如果logModule的lastEntry的term小于自己，也是不能投票的
			if lastEntry.Term > reqVoteParam.Term {
				reqVoteResult.IsVoted = false
				reqVoteResult.Term = con.Node.CurrentTerm
				return nil
			}
			// term符合的情况下，如果logModule的lastEntry的term也符合，但是index小于自己，也是不能投票的
			if lastEntry.Index > reqVoteParam.LastLogIndex {
				reqVoteResult.IsVoted = false
				reqVoteResult.Term = con.Node.CurrentTerm
				return nil
			}
		}
		// 成功投票并且设置leader为对方，即使对方选举失败，也会进入新的term,从而有新的心跳来更新自己的leader
		con.Node.State = node.FOLLOWER
		con.Node.LeaderAddr = reqVoteParam.ServiceId
		con.Node.CurrentTerm = reqVoteParam.Term
		con.Node.VotedFor = ""
		reqVoteResult.IsVoted = true
		reqVoteResult.Term = con.Node.CurrentTerm
		return nil
	}
	reqVoteResult.IsVoted = false
	reqVoteResult.Term = con.Node.CurrentTerm
	return nil
}
func (con *Consensus) HandlerAppendEntries(args *any, reply *any) error {
	appendEntriesArgs := &rpc.AppendEntriesArgs{}
	if err := mapstructure.Decode(*args, appendEntriesArgs); err != nil {
		return err
	}
	appendResult := &rpc.AppendResult{}
	if err := mapstructure.Decode(*reply, appendResult); err != nil {
		return err
	}
	Log.Infof("%s get a vote req ,args : %+v ", con.Node.SelfAddr, appendEntriesArgs)
	defer func() {
		*args = appendEntriesArgs
		*reply = appendResult
	}()
	if !con.appendLock.TryLock() {
		Log.Errorf("node %s get a appendLock failed", con.Node.SelfAddr)
		appendResult.Success = false
		appendResult.Term = con.Node.CurrentTerm
		return nil
	}
	defer con.appendLock.Unlock()
	if appendEntriesArgs.Term < con.Node.CurrentTerm {
		appendResult.Success = false
		appendResult.Term = con.Node.CurrentTerm
		return nil
	}
	// 更新心跳
	con.Node.PreHeartBeatTime = time.Now().Unix()
	// 只要发送了心跳说明当前周期还在持续，相当于延缓下次选举时间
	con.Node.PreElectionTime = time.Now().Unix()
	con.Node.LeaderAddr = appendEntriesArgs.LeaderId
	if appendEntriesArgs.Term >= con.Node.CurrentTerm {
		con.Node.State = node.FOLLOWER
		con.Node.CurrentTerm = appendEntriesArgs.Term
		Log.Infof("node : %s become follower , newTerm : %d , LeaderId : %s", con.Node.SelfAddr, appendEntriesArgs.Term, appendEntriesArgs.LeaderId)
	}
	//情况1：心跳 心跳不仅要更新心跳时间，也要根据leader的信息对自己的信息进行变更
	if appendEntriesArgs.Entries == nil || len(*appendEntriesArgs.Entries) == 0 {
		Log.Infof("leader %s send a heartbeat to node %s", appendEntriesArgs.LeaderId, con.Node.SelfAddr)

		nextCommit := con.Node.CommitIndex + 1
		//如果 leaderCommit > commitIndex，令 commitIndex 等于 leaderCommit 和 当前日志的最后一个Index
		// 意思就是当leaderCommit大于commitIndex时，就需要提交当前logModule中存在的所有log
		if appendEntriesArgs.LeaderCommit > con.Node.CommitIndex {
			floatMin := math.Min(float64(appendEntriesArgs.LeaderCommit), float64(con.Node.Config.LogModule.GetLastIndex()))
			min := int64(floatMin)
			con.Node.CommitIndex = min
			con.Node.LastApplied = min
		}
		// 更新之后的commitIndex如果大于下个要插入的位置，那就直接把nextCommit到commitIndex之间的所有log提交到状态机
		for ; nextCommit <= con.Node.CommitIndex; nextCommit++ {
			if success := con.Node.StateMachine.Set(con.Node.Config.LogModule.Get(nextCommit)); success {
				Log.Infof("node %s apply a entry to stateMachine successfully", con.Node.SelfAddr)
			} else {
				Log.Fatalf("node %s apply a entry to stateMachine failed", con.Node.SelfAddr)
			}
		}
		appendResult.Success = true
		appendResult.Term = con.Node.CurrentTerm
		return nil
	}
	// 情况2：真实日志 非空entries

	// 两种失败的情况：term不对或者term对但index不对(前提是已经存在一些log)，都需要上层减小index重试
	if con.Node.Config.LogModule.GetLastIndex() != 0 && appendEntriesArgs.PrevLogIndex != 0 {
		entry := con.Node.Config.LogModule.Get(appendEntriesArgs.PrevLogIndex)
		if entry != nil {
			// index存在时即index相同，此时如果term不相同，说明不是同一条日志，需要返回错误，并且让上层减小preIndex重试，直到找到符合的日志
			if entry.Term != appendEntriesArgs.Term {
				appendResult.Success = false
				appendResult.Term = con.Node.CurrentTerm
				return nil
			}
			// term 如果相同，那就是同一条日志，接下来就可以运行下面的成功处理的命令
		} else {
			// preLog处的日志为空，说明要添加的第一个日志的index前面为空，如果这样直接添加会造成空隙，所以需要将firstLog的index重试减小，直到preLogIndex处存在日志
			appendResult.Success = false
			appendResult.Term = con.Node.CurrentTerm
			return nil
		}
	}
	// 成功处理的情况：term符合，preIndex刚好落在本logModule的LastIndex上，可以从preLogIndex处拼接entries

	// 检测一下index相同但是term不同的情况：如果已经存在的日志条目和新的产生冲突（当前的preIndex+1已经存在entry了，索引值相同但是任期号不同），删除这一条和之后所有的.
	// 作用是避免覆盖式写入或者正常写入后仍然存在尾部的"脏数据"，所以将符合index、term要求后面的log完全删除
	uselessLog := con.Node.Config.LogModule.Get(appendEntriesArgs.PrevLogIndex + 1)
	if uselessLog != nil {
		if uselessLog.Term != (*appendEntriesArgs.Entries)[0].Term {
			err := con.Node.Config.LogModule.DeleteOnStartIndex(appendEntriesArgs.PrevLogIndex + 1)
			if err != nil {
				Log.Error(err)
				appendResult.Success = false
				appendResult.Term = con.Node.CurrentTerm
				return err
			}
		}
		// 第二个日志的位置已经存在相同的log了，说明之前已经写过一次了，防止重复写入
		appendResult.Success = true
		appendResult.Term = con.Node.CurrentTerm
		return nil
	}
	// 所以不符合条件的情况处理完成后
	// 写进日志 ，这不是提交，所以commitIndex不用变，等下一次发来心跳时再更新commitIndex,也就是说当leader发现过半收到新日志时，会在下一次发布心跳时携带更新后的commitIndex让follower同步
	for _, entry := range *appendEntriesArgs.Entries {
		con.Node.Config.LogModule.Write(&entry)
	}
	// 更新node信息
	// 下一个需要提交的日志的索引（如有）
	nextCommit := con.Node.CommitIndex + 1
	/*
		此时进行了新增entry操作，需要更新一下commitIndex : min(lastIndex,leaderCommit) 这里只是更新一下新的LastIndex<LeaderCommit的情况，
		而 LastIndex > leaderCommit的情况需要在leader统一提交时更新，也就是上边的那个相同操作
	*/
	// 如果 leaderCommit > commitIndex，令 commitIndex 等于 leaderCommit 和 当前日志的最后一个Index
	if appendEntriesArgs.LeaderCommit > con.Node.CommitIndex {
		floatMin := math.Min(float64(appendEntriesArgs.LeaderCommit), float64(con.Node.Config.LogModule.GetLastIndex()))
		min := int64(floatMin)
		con.Node.CommitIndex = min
		con.Node.LastApplied = min
	}
	// 此处的nextCommit代表未更新之前的commit，如果更新后的commit大于他，就需要将nextCommit之前的提交。依次提交直到nextCommit等于commitIndex + 1
	// 真正按照raft的原理应该不应该在这里直接提交，因为在这里就提交可能会无法处理leader的replication操作失败的情况，如果失败follower是不能提交的，可以等leader下次心跳时同步commit来判断是否需要提交;
	// 不过如果leader心跳前挂了，但是replication操作已经成功一般时，那么就无法完成提交了，这是一种取舍。
	// 在这种情况下即使提交了commitIndex前的数据，如果存在leader成功后但是宕机，也会有新的leader来更新，从而回退logModule，在之后新增log时也会更新掉stateMachine中的数据
	for ; nextCommit <= con.Node.CommitIndex; nextCommit++ {
		entry := con.Node.Config.LogModule.Get(nextCommit)
		success := con.Node.StateMachine.Set(entry)
		if success == false {
			Log.Fatalf("node %s apply logEntry{%+v} to stateMachine failed", con.Node.SelfAddr, entry)
		}
		nextCommit++
	}
	appendResult.Success = true
	appendResult.Term = con.Node.CurrentTerm
	con.Node.State = node.FOLLOWER
	return nil
}
