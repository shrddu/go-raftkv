package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "set a value",
	Long:  `set a value to servers`,
	Run: func(cmd *cobra.Command, args []string) {
		if args != nil {
			if len(args) > 1 {
				fmt.Printf("invaid command args")
			}
			//TODO 通过临时创建一个client 处理set命令，（调用client api进行存储） new client{}.
			return

		} else {
			fmt.Println("failed,please add a value behind the 'set'")
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
