package main

//var SM *statemachine.StateMachine
//var port = flag.String("port", "9990", "server listen port")

type Reply interface{}
type ClientRPCReply struct {
	Success bool
}

func main() {

	//flag.Parse()
	//s := server.NewServer()
	//s.RegisterName("Handler", new(rpc.No), "")
	//err := s.Serve("tcp", "localhost:"+*port)
	//if err != nil {
	//	panic(err)
	////}
	//var reply Reply = &ClientRPCReply{Success: true}
	//
	//reply := (*Reply)(unsafe.Pointer(clientReply))
	//fmt.Printf("%+v", reply)
}
