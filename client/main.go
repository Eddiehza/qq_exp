package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"exp/proto"
)

func main() {

	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}

	var msg proto.Msg
	var user_id uint32
	login_status := false
	input := bufio.NewReader(os.Stdin)

	//开始登陆
	for !login_status {
		fmt.Println("输入账号密码")

		//账号
		s, _ := input.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.ToUpper(s) == "Q" {
			return
		}
		msg1 := []byte(s)
		msg.Write(conn, 1, 1, msg1, 1)

		//密码
		s, _ = input.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.ToUpper(s) == "Q" {
			return
		}
		msg1 = []byte(s)
		msg.Write(conn, 0, 0, msg1, 1)

		msg.Read(conn)
		if !login_status && msg.Sender == 0 && msg.Flags == 2 {
			fmt.Println("登陆成功")
			tempId, _ := strconv.Atoi(string(msg.Data))
			user_id = uint32(tempId)
			fmt.Println("用户id:", user_id)
			login_status = true
		} else if !login_status && msg.Sender == 0 && msg.Flags == 3 {
			fmt.Println("登陆失败")
			fmt.Println(string(msg.Data))
		}
	}

	go func() {
		for {
			msg.Read(conn)
			if msg.Sender == 0 {
				fmt.Printf("系统信息：%v\n", string(msg.Data))
			} else if msg.Flags == 11 {
				fmt.Println(string(msg.Data))
			}
		}
	}()

	for {
		s, _ := input.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.ToUpper(s) == "Q" {
			return
		}

		msg1 := []byte(s)
		msg.Write(conn, user_id, proto.User2.Id, msg1, 11)

	}
}
