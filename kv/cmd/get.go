package cmd

import (
	"fmt"
	"go-raftkv/common/rpc"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get the value of the key you input",
	Long:  `get the value of the key you input`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		for _, addr := range addrs {
			request := &rpc.MyRequest{
				RequestType:   rpc.CLIENT_REQ,
				ServiceId:     addr,
				ServicePath:   "Node",
				ServiceMethod: "HandlerClientRequest",
				Args: &rpc.ClientRPCArgs{
					Tp: rpc.CLIENT_REQ_TYPE_GET,
					K:  key,
				},
			}
			client := &rpc.RpcClient{}
			reply := client.Send(request)
			rpcReply := reply.(*rpc.ClientRPCReply)
			if rpcReply.Success {
				fmt.Printf("get successfully , the value of '%s' is : %s \n", key, rpcReply.Entry.V)
				break
			} else {
				fmt.Println("get failed")
				break
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
