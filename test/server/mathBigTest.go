/*
*

	@author: 18418
	@date: 2023/10/9
	@desc:

*
*/
package main

import (
	"fmt"
	"math/big"
)

func main() {
	var index int64 = 0
	b := &big.Int{}
	intPointer := b.SetInt64(index)
	c := &big.Int{}
	bytes := intPointer.Bytes()
	fmt.Println(bytes)
	c.SetBytes(bytes)
	fmt.Println(c.Int64())
}
