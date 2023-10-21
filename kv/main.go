/*
Copyright © 2023 shr@1841892879@qq.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import "go-raftkv/kv/cmd"

/* 需要手动 go install 此程序为exe文件到 GOPATH/bin 目录下，这样就可以在 terminal 里用 (kv set key111 value111 或者kv get key111) 来操作数据库 */
func main() {
	cmd.Execute()
}
