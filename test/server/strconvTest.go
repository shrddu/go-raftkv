/*
*

	@author: 18418
	@date: 2023/10/10
	@desc:

*
*/
package main

import (
	"fmt"
	"strconv"
)

func main() {
	s := "1523146"
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {

	}
	fmt.Println(i)
}
