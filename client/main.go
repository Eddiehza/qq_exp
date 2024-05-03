package main

import (
	"bufio"
	"crypto/tls"
	"exp/proto"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var file_save_path string

func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	file_save_path = currentPath + "/public/client"

	input := bufio.NewReader(os.Stdin)
	fmt.Println("服务器地址：")
	targetIP := ReadFromBufWithoutExit(input)
	if targetIP == "" {
		targetIP = "127.0.0.1:9091"
	}

	fmt.Println(targetIP)

	// 创建一个TLS配置
	config := &tls.Config{
		InsecureSkipVerify: true,
	}

	// 连接到服务器
	conn, err := tls.Dial("tcp", targetIP, config)
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
			if msg.Flags == proto.FLAG_PORT {
				port, err := strconv.Atoi(string(msg.Data))
				if err != nil {
					log.Fatal(err)
				}

				// 创建一个UDP地址，使用所有网络接口
				udpAddr := &net.UDPAddr{
					IP:   net.IPv4zero,
					Port: port,
				}

				// 在该地址上创建一个UDP监听器
				udpListener, err := net.ListenUDP("udp", udpAddr)
				if err != nil {
					log.Fatal(err)
				}
				defer udpListener.Close()
				fmt.Println(udpAddr)
				// 获取实际监听的端口号
				actualPort := udpListener.LocalAddr().(*net.UDPAddr).Port
				fmt.Printf("正在监听UDP端口 %d\n", actualPort)

				// 在新的goroutine中处理UDP连接
				go func() {
					for {
						buf := make([]byte, 1024)
						n, addr, err := udpListener.ReadFromUDP(buf)
						if err != nil {
							log.Printf("Error reading from UDP: %v\n", err)
							continue
						}
						fmt.Printf("收到来自 %s 的消息: %s\n", addr, string(buf[:n]))

						// 发送确认消息
						ack := []byte("ACK")
						_, err = udpListener.WriteToUDP(ack, addr)
						if err != nil {
							log.Printf("Error sending ACK: %v\n", err)
						}
					}
				}()

			} else if msg.Flags == proto.FLAG_P2P {
				// 服务器返回目标客户端的公网地址和端口
				targetAddr := string(msg.Data)
				// 解析目标地址
				udpAddr, err := net.ResolveUDPAddr("udp", targetAddr)
				if err != nil {
					fmt.Println("解析地址失败:", err)
					return
				}

				// 建立UDP连接
				conn, err := net.DialUDP("udp", nil, udpAddr)
				if err != nil {
					fmt.Println("建立连接失败:", err)
					return
				}
				defer conn.Close()

				fmt.Printf("与 %s 建立UDP连接\n", targetAddr)
				// 要发送的消息
				message := []byte("Hello, World!")
				// 重试次数
				retries := 100

				for i := 0; i < retries; i++ {
					// 发送消息
					n, err := conn.Write(message)
					if err != nil {
						fmt.Println("发送消息失败:", err)
						return
					}

					if n != len(message) {
						fmt.Println("未能发送完整的消息")
						return
					}

					fmt.Printf("已向 %s 发送消息\n", targetAddr)

					// 等待确认消息
					buf := make([]byte, 1024)
					conn.SetReadDeadline(time.Now().Add(time.Second * 5)) // 等待5秒
					n, _, err = conn.ReadFromUDP(buf)
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							fmt.Println("确认消息超时，重试发送")
							continue
						} else {
							fmt.Println("读取确认消息失败:", err)
							return
						}
					}

					// 检查确认消息
					if string(buf[:n]) == "ACK" {
						fmt.Println("消息已确认")
						break
					}
				}
			} else if msg.Sender == 0 && msg.Flags == proto.FLAG_FILE {
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
			if strings.HasPrefix(sendMsg, "p2p") {
				// 分割字符串并获取用户ID
				parts := strings.Split(sendMsg, " ")
				if len(parts) < 2 {
					fmt.Println("请输入正确的命令，例如：p2p 2")
					continue
				}
				targetUserId := strings.TrimSpace(parts[1])
				// 向服务器发送一个特殊的p2p消息
				msg.Write(conn, 1, receiver_id, []byte(targetUserId), proto.FLAG_P2P)
			} else if strings.HasPrefix(sendMsg, "sendfile") {
				fields := strings.Fields(sendMsg)
				if len(fields) > 1 && fields[0] == "sendfile" {
					filePath := strings.Join(fields[1:], " ")
					// 去除路径两端的双引号
					filePath = strings.Trim(filePath, "\"")
					go file.Send(conn, 1, receiver_id, filePath)
				}
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
