package node

import (
	"context"
	_interface "go-raftkv/common/interface"
)

type INode interface {
	_interface.ILifecycle
	//
	// SetConfig
	//  @Description: 设置配置
	//  @param NodeConfig
	//
	SetConfig(Config)

	//
	// HandlerRequestVote
	//  @Description: 处理投票请求
	//  @param ctx
	//  @param args
	//  @param reply
	//  @return error
	//
	HandlerRequestVote(ctx context.Context, args *any, reply *any) error
	//
	// HandlerAppendEntries
	//  @Description: 处理附加日志请求
	//  @param ctx
	//  @param args
	//  @param reply
	//  @return error
	//
	HandlerAppendEntries(ctx context.Context, args *any, reply *any) error
	//
	// HandlerClientRequest
	//  @Description: 处理客户端的kv请求
	//  @param ctx
	//  @param args
	//  @param reply
	//  @return error
	//
	HandlerClientRequest(ctx context.Context, args *any, reply *any) error
	//
	// Redirect
	//  @Description: 当接收到client的set请求时，如果自己不是leader，需要转发给leader，并且接收到leader的返回值后，返回给client结果
	//  @param ctx
	//  @param args
	//  @param reply
	//  @return error
	//
	Redirect(ctx context.Context, args *any, reply *any) error
}
