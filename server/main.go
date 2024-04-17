package main

import (
	"fmt"
	"net"

	"exp/proto"
)

func process(conn net.Conn) {
	// 处理完关闭连接
	defer conn.Close()
	// 针对当前连接做发送和接受操作
	var msg proto.Msg
	msg.Read(conn)

	msg2 := []byte("qs")
	msg.Write(conn, proto.User2, proto.User1, msg2)
	//将接受到的数据返回给客户端
}

func main() {
	// 建立 tcp 服务
	listen, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}

	for {
		// 等待客户端建立连接
		conn, err := listen.Accept()
		if err != nil {
			fmt.Printf("accept failed, err:%v\n", err)
			continue
		}
		// 启动一个单独的 goroutine 去处理连接
		go process(conn)
	}
}
