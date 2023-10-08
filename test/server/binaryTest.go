/*
*

	@author: 18418
	@date: 2023/9/27
	@desc:

*
*/
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type demo struct {
	I int64
	S string
}

func main() {
	data := &demo{
		I: 111,
		S: "value",
	}
	buf := &bytes.Buffer{}
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf.Bytes())
	decoder := gob.NewDecoder(buf)
	var dataDecode *demo
	err = decoder.Decode(&dataDecode) //需要传地址
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v", dataDecode)
}
