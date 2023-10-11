package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go-raftkv/common/rpc"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "set a value",
	Long:  `set a value to servers`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		for _, addr := range addrs {
			request := &rpc.MyRequest{
				RequestType:   rpc.CLIENT_REQ,
				ServiceId:     addr,
				ServicePath:   "Node",
				ServiceMethod: "HandlerClientRequest",
				Args: &rpc.ClientRPCArgs{
					Tp: rpc.CLIENT_REQ_TYPE_SET,
					K:  key,
					V:  value,
				},
			}
			client := &rpc.RpcClient{}
			reply := client.Send(request)
			rpcReply := reply.(*rpc.ClientRPCReply)
			if rpcReply.Success {
				fmt.Println("set successfully")
				break
			} else {
				fmt.Println("set failed")
				break
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
