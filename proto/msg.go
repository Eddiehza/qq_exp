package proto

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/schollz/progressbar/v3"
)

const HEAD_LEN = 18

const (
	FLAG_RESERVE = iota
	FLAG_LOGIN
	FLAG_SUCCESS
	FLAG_FAILURE
	FLAG_UNREACHABLE
	FLAG_DISCONNECT

	FLAG_TEXT = iota + 11
	FLAG_FILE
	FLAG_P2P
	FLAG_PORT
)

type Msg struct {
	Length uint64 // 8
	//消息类型
	//0~10是系统消息，1是登陆尝试，2是登陆成功，3是登陆失败，4是对方未登陆提示，5是用户断开链接
	//11是普通文字消息
	Flags uint8 // 1
	//若同一个人同时发送多个包，可以使用此序号代表文件
	Extra_info uint8  // 1
	Sender     uint32 // 4
	Receiver   uint32 // 4
	Data       []byte // 4
}

func (msg Msg) Marshal() []byte {
	buf := make([]byte, msg.Length)
	binary.LittleEndian.PutUint64(buf, msg.Length)
	buf[8] = byte(msg.Flags)
	buf[9] = byte(msg.Extra_info)
	binary.LittleEndian.PutUint32(buf[10:14], msg.Sender)
	binary.LittleEndian.PutUint32(buf[14:18], msg.Receiver)
	copy(buf[HEAD_LEN:], msg.Data)

	return buf
}

func (msg *Msg) Unmarshal(msg_byte []byte) {
	msg.Length = binary.LittleEndian.Uint64(msg_byte[0:8])
	msg.Flags = msg_byte[8]
	msg.Extra_info = msg_byte[9]
	msg.Sender = binary.LittleEndian.Uint32(msg_byte[10:14])
	msg.Receiver = binary.LittleEndian.Uint32(msg_byte[14:18])
	msg.Data = make([]byte, msg.Length-HEAD_LEN)
	copy(msg.Data, msg_byte[HEAD_LEN:])
}

func (msg *Msg) Write(conn net.Conn, senderId uint32, receiverId uint32, data []byte, flag uint8) {
	msg.Length = uint64(HEAD_LEN + len(data))
	msg.Flags = flag
	//msg.Extra_info
	// if flag == FILE  Extra_info = len of File Name
	msg.Sender = senderId
	msg.Receiver = receiverId
	msg.Data = data
	marshal_msg := msg.Marshal()
	// fmt.Printf("发送数据：%+v\n", msg)

	if flag == FLAG_FILE {
		fmt.Println("开始传输文件")
		totalBytes := len(marshal_msg)
		percent := totalBytes / 100
		bar := progressbar.Default(100)
		for i := 0; i < 100; i++ {
			start := int64(i) * int64(percent)
			end := start + int64(percent)
			if i == 99 {
				end = int64(totalBytes)
			}

			partData := marshal_msg[start:end]

			_, err := conn.Write(partData)
			if err != nil {
				fmt.Printf("send failed, err:%v\n", err)
				return
			}

			bar.Add(1)
		}

	} else {
		_, err := conn.Write(marshal_msg)
		if err != nil {
			fmt.Printf("send failed, err:%v\n", err)
			return
		}
	}

}

func (msg *Msg) Read(conn net.Conn) bool {
	// reader := bufio.NewReader(conn)
	var buf []byte = make([]byte, 8)
	for {
		// var err error
		// buf, err = reader.Peek(8)
		_, err := io.ReadFull(conn, buf)
		// fmt.Println("buf为:", buf)
		// fmt.Println("转换后:", binary.LittleEndian.Uint64(buf))
		if err == io.EOF {
			return false
		}
		if err == nil {
			break
		}
	}
	length := binary.LittleEndian.Uint64(buf)
	complete_msg := make([]byte, length)
	for {
		_, err := io.ReadFull(conn, complete_msg[8:])
		//need a timeout
		if err == nil {
			break
		}
	}
	copy(complete_msg[:8], buf)
	// fmt.Println(complete_msg)
	msg.Unmarshal(complete_msg)
	// fmt.Printf("收到的数据：%+v\n", msg)
	return true
}
