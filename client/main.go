package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"exp/proto"

	_ "github.com/mattn/go-sqlite3"
)

var file_save_path string

type msg_abstract struct {
	Sender    uint32
	Receiver  uint32
	Content   string
	CreatedAt string
}

func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	file_save_path = currentPath + "/public/client"

	input := bufio.NewReader(os.Stdin)
	targetIP := ReadFromBufWithoutExit(input)
	if targetIP == "" {
		targetIP = "127.0.0.1:9091"
	}

	// fmt.Println(targetIP)

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

		//账号
		msg.Write(conn, 0, 0, []byte(ReadFromBufWithoutExit(input)), proto.FLAG_LOGIN)

		//密码
		msg.Write(conn, 0, 0, []byte(ReadFromBufWithoutExit(input)), proto.FLAG_LOGIN)

		msg.Read(conn)
		if !login_status && msg.Sender == 0 && msg.Flags == proto.FLAG_SUCCESS {
			fmt.Println("success")
			tempId, _ := strconv.Atoi(string(msg.Data))
			user_id = uint32(tempId)

			//退出前给服务端发送通知，之后删
			defer msg.Write(conn, user_id, proto.Server.Id, []byte(""), proto.FLAG_DISCONNECT)

			login_status = true
		} else if !login_status && msg.Sender == 0 && msg.Flags == proto.FLAG_FAILURE {
			fmt.Println(string(msg.Data))
		}
	}

	//查询用户的本地记录
	db, err := sql.Open("sqlite3", file_save_path+"/client.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 创建表
	// fmt.Printf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY,receiver TEXT,sender TEXT,message TEXT,createdAt DATETIME)", "user"+strconv.FormatUint(uint64(user_id), 10))
	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY,receiver TEXT,sender TEXT,message TEXT,createdAt DATETIME)", "user"+strconv.FormatUint(uint64(user_id), 10)))
	if err != nil {
		panic(err)
	}

	//查询有没有离线消息，有则转发
	rows, err := db.Query(fmt.Sprintf("SELECT receiver, sender, message, createdAt FROM %s", "user"+strconv.FormatUint(uint64(user_id), 10)))
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var local_history_msgs []msg_abstract

	for rows.Next() {
		var history_msg msg_abstract
		err := rows.Scan(&history_msg.Receiver, &history_msg.Sender, &history_msg.Content, &history_msg.CreatedAt)
		if err != nil {
			panic(err)
		}
		local_history_msgs = append(local_history_msgs, history_msg)
	}
	jsonData, err := json.Marshal(local_history_msgs)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return
	}
	fmt.Printf("local-history-msg:%v\n", string(jsonData))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	//记录通信方账号
	// tempId, _ := strconv.Atoi(ReadFromBufWithoutExit(input))
	// receiver_id = uint32(tempId)

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
			} else if msg.Flags == proto.FLAG_FRIEND_LIST {
				var uint32Slice []uint32
				for i := 0; i < len(msg.Data); i += 4 {
					val := binary.LittleEndian.Uint32(msg.Data[i : i+4])
					uint32Slice = append(uint32Slice, val)
				}
				fmt.Printf("friend-list:%v\n", uint32Slice)
			} else if msg.Flags == proto.FLAG_HISTORY_MSG {
				fmt.Printf("history-msg:%v\n", string(msg.Data))
				var messages []msg_abstract
				err := json.Unmarshal(msg.Data, &messages)
				if err != nil {
					fmt.Println("解析历史消息失败:", err)
					return
				}
				for _, singleMsg := range messages {
					query := "INSERT INTO " + "user" + strconv.FormatUint(uint64(user_id), 10) + " (receiver, sender, message, createdAt) VALUES (?, ?, ?, ?)"
					_, err := db.Exec(query, singleMsg.Receiver, singleMsg.Sender, singleMsg.Content, singleMsg.CreatedAt)
					if err != nil {
						panic(err)
					}
				}
			} else if msg.Sender == 0 {
				fmt.Printf("系统信息：%v\n", string(msg.Data))
			} else if msg.Flags == proto.FLAG_TEXT {
				fmt.Printf("text-sender:%v %v\n", msg.Sender, string(msg.Data))
				//存入数据库里
				currentTime := time.Now()
				createdAt := currentTime.Format("2006-01-02 15:04:05")
				query := "INSERT INTO " + "user" + strconv.FormatUint(uint64(user_id), 10) + " (receiver, sender, message, createdAt) VALUES (?, ?, ?, ?)"
				_, err := db.Exec(query, msg.Receiver, msg.Sender, msg.Data, createdAt)
				if err != nil {
					panic(err)
				}
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
			if strings.HasPrefix(sendMsg, "setaim") {
				setaim := strings.Fields(sendMsg)
				receiver_64id, err := strconv.ParseUint(setaim[1], 10, 32)
				if err != nil {
					fmt.Println("Error converting string to uint32:", err)
					return
				}
				receiver_id = uint32(receiver_64id)
			} else if strings.HasPrefix(sendMsg, "sendfile") {
				fields := strings.Fields(sendMsg)
				if len(fields) > 1 && fields[0] == "sendfile" {
					filePath := strings.Join(fields[1:], " ")
					// 去除路径两端的双引号
					filePath = strings.Trim(filePath, "\"")
					go file.Send(conn, user_id, receiver_id, filePath)
				}
			} else {
				var msg proto.Msg
				msg.Write(conn, user_id, receiver_id, []byte(sendMsg), proto.FLAG_TEXT)
				currentTime := time.Now()
				createdAt := currentTime.Format("2006-01-02 15:04:05")
				query := "INSERT INTO " + "user" + strconv.FormatUint(uint64(user_id), 10) + " (receiver, sender, message, createdAt) VALUES (?, ?, ?, ?)"
				_, err := db.Exec(query, msg.Receiver, msg.Sender, msg.Data, createdAt)
				if err != nil {
					panic(err)
				}
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
