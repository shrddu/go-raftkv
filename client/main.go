/*
*

	@author: 18418
	@date: 2023/10/20
	@desc:

*
*/
package main

import (
	"github.com/smallnest/rpcx/log"
	"go-raftkv/client/clientMethod"
)

func main() {
	for i := 1; i <= 100; i++ {
		isSuccess := clientMethod.SendASetRequest("key1", "value1")
		if isSuccess {
			clientMethod.Log.Infof("第 %d 次 send set success", i)
		} else {
			log.Infof("send set failed")
		}
	}
}
