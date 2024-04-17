package proto

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const HEAD_LEN = 30

type Msg struct {
	Length uint64 // 8
	Flags  uint8  // 1
	Id     uint8  // 1
	//若同一个人同时发送多个包，可以使用此序号代表文件
	Sender   [10]byte // 10
	Receiver [10]byte // 10
	Data     []byte   // 4
}

func (msg Msg) Marshal() []byte {
	buf := make([]byte, msg.Length)
	binary.LittleEndian.PutUint64(buf, msg.Length)
	buf[8] = byte(msg.Flags)
	buf[9] = byte(msg.Id)
	copy(buf[10:20], msg.Sender[:])
	copy(buf[20:30], msg.Receiver[:])
	copy(buf[30:], msg.Data)

	return buf
}

func (msg *Msg) Unmarshal(msg_byte []byte) {
	msg.Length = binary.LittleEndian.Uint64(msg_byte[0:8])
	msg.Flags = msg_byte[8]
	msg.Id = msg_byte[9]
	copy(msg.Sender[:], msg_byte[10:20])
	copy(msg.Receiver[:], msg_byte[20:30])
	msg.Data = make([]byte, msg.Length-HEAD_LEN)
	copy(msg.Data, msg_byte[30:])
}

func (msg *Msg) Write(conn net.Conn, sender User, receiver User, data []byte) {
	msg.Length = uint64(HEAD_LEN + len(data))
	msg.Flags = 32 //随便设置的，之后改
	msg.Id = 1     //随便设置的，之后改
	msg.Sender = sender.Id
	msg.Receiver = receiver.Id
	msg.Data = data
	marshal_msg := msg.Marshal()
	fmt.Printf("发送数据：%+v\n", msg)
	_, err := conn.Write(marshal_msg)
	if err != nil {
		fmt.Printf("send failed, err:%v\n", err)
		return
	}
}

func (msg *Msg) Read(conn net.Conn) {
	reader := bufio.NewReader(conn)
	buf, err := reader.Peek(8)
	if err != nil {
		fmt.Printf("read from conn failed, err:%v\n", err)
		return
	}
	length := binary.LittleEndian.Uint64(buf)
	complete_msg := make([]byte, length)
	for {
		_, err := io.ReadFull(reader, complete_msg)
		//need a timeout
		if err == nil {
			break
		}
	}
	msg.Unmarshal(complete_msg)
	fmt.Printf("收到的数据：%+v\n", msg)
	fmt.Printf("收到的消息：%v\n", string(msg.Data))
}
