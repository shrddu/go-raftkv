/*
*

	@author: 18418
	@date: 2023/10/21
	@desc:

*
*/
package node

type NodeState struct {

	/** 已知的最大的已经被提交的LogModule日志条目的索引值 */
	CommitIndex int64

	/** 最后被应用到状态机的日志条目索引值（初始化为 0，持续递增) */
	LastApplied int64

	/** 对于每一个服务器，需要发送给他的下一个日志条目的索引值（初始化为领导人最后索引值加一） 不仅初始化时要设置，当follower变成leader时也需要重新设置*/
	NextIndexs map[string]int64 // map[serverAddr]nextIndex

	/** 对于每一个服务器，已经复制给他的日志的最高索引值 */
	CopiedIndexs map[string]int64 // map[serverAddr]nextIndex
}
