package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"exp/proto"
	"sync"
)

var user_tcp_chat sync.Map
var offline_files sync.Map
var file_save_path string

type file_abstract struct {
	file_path    string
	senderId     uint32
	expired_time time.Time
	received     bool
}

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

	_, ok := user_tcp_chat.Load(user_id)
	if !ok {
		user_tcp_chat.Store(user_id, conn)
		defer user_tcp_chat.Delete(user_id)
	}

	//用户上线时检查是否有离线的文件，有则发送
	value, ok := offline_files.Load(user_id)
	if ok {
		fmt.Println(value)
		for _, offline_file := range value.([]file_abstract) {
			fmt.Println(offline_file.file_path)
			fmt.Println(offline_file.senderId)
			go file.Send(conn, offline_file.senderId, user_id, fmt.Sprintf("%v/%v", file_save_path, offline_file.file_path))
			offline_file.received = true
		}
		offline_files.Delete(user_id)
	}

	go func() {
		ticker := time.NewTicker(time.Minute) // 每分钟检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				offline_files.Range(func(key, value interface{}) bool {
					files, ok := value.([]file_abstract)
					if !ok {
						return true
					}

					// 检查每个文件是否过期
					for i, file := range files {
						if time.Now().After(file.expired_time) {
							// 如果文件过期，从 offline_files 中删除
							files = append(files[:i], files[i+1:]...)
							offline_files.Store(key, files)
							if file.received {
								// 从文件系统中删除文件
								os.Remove(filepath.Join(file_save_path, file.file_path))
							} else if !file.received {
								// 通知发送方文件未接收
								if senderConn, ok := user_tcp_chat.Load(file.senderId); ok {
									if conn, ok := senderConn.(net.Conn); ok {
										msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte("对方未接收文件"), proto.FLAG_UNREACHABLE)
										os.Remove(filepath.Join(file_save_path, file.file_path))
									}
								} else if !ok {
									// 如果发送方不在线,保留文件
								}

							}

						}
					}

					return true
				})
			}
		}
	}()
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

				// 构造文件已保存的确认消息，包括文件名
				//confirmationMsg := fmt.Sprintf("文件已保存到: %s", fileName)

				if receiverConn, ok := user_tcp_chat.Load(msg.Receiver); ok {
					if conn, ok := receiverConn.(net.Conn); ok {
						msg.Write(conn, msg.Sender, msg.Receiver, msg.Data, proto.FLAG_FILE)
						fmt.Printf("服务器转发文件到客户端 %v\n", msg.Receiver)
					}
				} else {
					fileName, err := file.Receive(msg, file_save_path, true)
					if err != nil {
						log.Printf("Error receiving file: %v\n", err)
						msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte("文件接收失败"), proto.FLAG_FAILURE)
						return
					}
					confirmationMsg := fmt.Sprintf("对方未登录！%s已保存到服务器", filepath.Base(fileName))

					new_file := file_abstract{
						file_path:    filepath.Base(fileName),
						senderId:     msg.Sender,
						expired_time: time.Now().Add(time.Hour * 24),
						received:     false,
					}
					value, ok := offline_files.Load(msg.Receiver)
					if !ok {
						var save_files []file_abstract
						save_files = append(save_files, new_file)
						offline_files.Store(msg.Receiver, save_files)
					} else {
						existing_files, ok := value.([]file_abstract)
						if !ok {
							// 处理类型断言失败的情况
							fmt.Println("转换失败")
							return
						}

						// 将文件路径添加到现有的数组中
						existing_files = append(existing_files, new_file)
						offline_files.Store(msg.Receiver, existing_files)
					}

					msg.Write(conn, proto.Server.Id, proto.Server.Id, []byte(confirmationMsg), proto.FLAG_UNREACHABLE)
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

	currentPath, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	file_save_path = currentPath + "/public/server"
	fmt.Println(file_save_path)

	// 加载服务器的证书和私钥
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}}

	// 创建一个TLS监听器
	listen, err := tls.Listen("tcp", "0.0.0.0:9091", &config)
	if err != nil {
		log.Fatalf("server: listen: %s", err)
	}
	log.Print("server: listening")

	port := 9091

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
