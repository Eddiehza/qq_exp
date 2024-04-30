package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"exp/proto"
)

var file_save_path = "./public/client"

func main() {

	input := bufio.NewReader(os.Stdin)
	fmt.Println("服务器地址：")
	targetIP := ReadFromBufWithoutExit(input)
	if targetIP == "" {
		targetIP = "127.0.0.1:9091"
	}

	fmt.Println(targetIP)

	conn, err := net.Dial("tcp", targetIP)
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}

	defer conn.Close()
	var msg proto.Msg
	var file proto.File
	var user_id uint32
	var receiver_id uint32
	login_status := false
	msgs := make(chan string, 1)

	//开始登陆
	for !login_status {
		fmt.Println("输入账号密码")

		//账号
		msg.Write(conn, 1, 1, []byte(ReadFromBufWithoutExit(input)), proto.FLAG_LOGIN)

		//密码
		msg.Write(conn, 0, 0, []byte(ReadFromBufWithoutExit(input)), proto.FLAG_LOGIN)

		msg.Read(conn)
		if !login_status && msg.Sender == 0 && msg.Flags == proto.FLAG_SUCCESS {
			fmt.Println("登陆成功")
			tempId, _ := strconv.Atoi(string(msg.Data))
			user_id = uint32(tempId)

			//退出前给服务端发送通知，之后删
			defer msg.Write(conn, user_id, proto.Server.Id, []byte(""), proto.FLAG_DISCONNECT)

			fmt.Println("用户id:", user_id)
			login_status = true
		} else if !login_status && msg.Sender == 0 && msg.Flags == proto.FLAG_FAILURE {
			fmt.Println("登陆失败")
			fmt.Println(string(msg.Data))
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	//记录通信方账号
	fmt.Println("通信方账号：")
	tempId, _ := strconv.Atoi(ReadFromBufWithoutExit(input))
	receiver_id = uint32(tempId)

	//监听来自服务器消息
	go func() {
		for {
			exit := msg.Read(conn)
			if !exit {
				sigs <- syscall.SIGINT
				fmt.Println("已与服务器断开连接")
				return
			}
			if msg.Sender == 0 && msg.Flags == proto.FLAG_FILE {
				_, err := file.Receive(msg, file_save_path, false)
				if err != nil {
					log.Printf("Error receiving file: %v\n", err)
				}
			} else if msg.Sender == 0 {
				fmt.Printf("系统信息：%v\n", string(msg.Data))
			} else if msg.Flags == proto.FLAG_TEXT {
				fmt.Println(string(msg.Data))
			} else if msg.Flags == proto.FLAG_FILE {
				_, err := file.Receive(msg, file_save_path, false)
				//fmt.Println(string((msg.Data)))
				if err != nil {
					log.Printf("Error receiving file: %v\n", err)
				}
			}
		}
	}()

	//开启键盘监听
	go ReadFromBuf(input, msgs)

	//主协程发送消息、处理错误
	for {
		select {
		case sendMsg, ok := <-msgs:
			if !ok {
				return
			}
			if strings.HasPrefix(sendMsg, "sendfile ") {
				filePath := strings.TrimPrefix(sendMsg, "sendfile ")
				go file.Send(conn, 1, receiver_id, filePath) // Assuming sendFile is defined
			} else {
				var msg proto.Msg
				msg.Write(conn, 1, receiver_id, []byte(sendMsg), proto.FLAG_TEXT)
			}
		case <-sigs:
			fmt.Println("收到中断信号")
			return
		}
	}
}

func ReadFromBufWithoutExit(input *bufio.Reader) string {
	s, _ := input.ReadString('\n')
	s = strings.TrimSpace(s)
	return s
}

func ReadFromBuf(input *bufio.Reader, msgs chan string) {
	for {
		s, _ := input.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.ToUpper(s) == "Q" {
			close(msgs)
			return
		} else {
			msgs <- s
		}
	}
}
