package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"exp/proto"
	"sync"
)

var user_tcp_chat sync.Map

func process(ctx context.Context, conn net.Conn) {
	// 处理完关闭连接
	defer conn.Close()
	var msg proto.Msg
	var user proto.User
	var file proto.File
	var user_id uint32
	for {
		exit := msg.Read(conn)
		if !exit {
			return
		}
		tempId, err := strconv.Atoi(string(msg.Data))
		if err != nil {
			msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte("输入有误"), proto.FLAG_FAILURE)
		}
		user.Id = uint32(tempId)
		msg.Read(conn)
		user.Passwd = string(msg.Data)

		fmt.Printf("登陆用户：%+v\n", user)
		status, logs := proto.Login(user)
		if !status {
			msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte(logs), proto.FLAG_FAILURE)
		} else {
			user_id = user.Id
			msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte(strconv.Itoa(int(user_id))), proto.FLAG_SUCCESS)
			break
		}
	}
	//用户上线时可能需要传递离线的文件等等

	_, ok := user_tcp_chat.Load(user_id)
	if !ok {
		user_tcp_chat.Store(user_id, conn)
		defer user_tcp_chat.Delete(user_id)
	}

	// 针对当前连接做发送和接受操作
	for {
		select {
		case <-ctx.Done():
			return
		default:
			exit := msg.Read(conn)
			if !exit {
				return
			}
			switch msg.Flags {
			case proto.FLAG_DISCONNECT:
				fmt.Println(msg.Sender, "断开连接")
				return
			case proto.FLAG_FILE:
				fileName, err := file.Receive(msg)
				if err != nil {
					log.Printf("Error receiving file: %v\n", err)
					msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte("文件接收失败"), proto.FLAG_FAILURE)
					return
				}

				// 构造文件已保存的确认消息，包括文件名
				confirmationMsg := fmt.Sprintf("文件已保存到: %s", fileName)

				if receiverConn, ok := user_tcp_chat.Load(msg.Receiver); ok {
					if conn, ok := receiverConn.(net.Conn); ok {
						msg.Write(conn, msg.Sender, msg.Receiver, []byte(confirmationMsg), proto.FLAG_FILE)
					}
				} else {
					msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte(confirmationMsg), proto.FLAG_FILE)
				}

			case proto.FLAG_TEXT:
				if receiver_conn, ok := user_tcp_chat.Load(msg.Receiver); ok {
					if receiver_conn, ok := receiver_conn.(net.Conn); ok {
						msg.Write(receiver_conn, msg.Sender, msg.Receiver, msg.Data, proto.FLAG_TEXT)
					}
				} else {
					msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte("对方未登录！"), proto.FLAG_UNREACHABLE)
				}
			}
		}
	}

}

func main() {
	port := 9091
	// 建立 tcp 服务
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", port))
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil && ip.IsGlobalUnicast() {
			fmt.Println("IP Address:", ip, ":", port)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	for {
		// 等待客户端建立连接
		conn, err := listen.Accept()
		if err != nil {
			fmt.Printf("accept failed, err:%v\n", err)
			continue
		}
		// 启动一个单独的 goroutine 去处理连接

		go process(ctx, conn)
	}
}
