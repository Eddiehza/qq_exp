package main

import (
	"fmt"
	"net"

	"exp/proto"
)

func main() {

	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}

	var msg proto.Msg

	msg1 := []byte("xybzsb")
	msg.Write(conn, proto.User1, proto.User2, msg1)

	msg.Read(conn)
}
