package proto

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

type File struct {
}

func (f File) Receive(msg Msg) (string, error) {
	// 构造文件名，这里使用发送者ID和".dat"后缀
	filename := "received_" + strconv.Itoa(int(msg.Sender)) + ".dat"

	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		log.Println("创建文件失败:", err)
		return "", err // 返回空文件名和错误信息
	}
	defer file.Close()

	// 将接收到的数据写入文件
	_, err = file.Write(msg.Data)
	if err != nil {
		log.Println("写入文件失败:", err)
		return filename, err // 返回文件名和错误信息
	}

	// 成功，返回文件名和nil表示没有错误
	fmt.Println("文件接收并保存为:", filename)
	return filename, nil
}

func (f File) Send(conn net.Conn, senderId uint32, receiverId uint32, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("无法获得文件信息:", err)
		return
	}

	// 读取文件内容到缓冲区
	data := make([]byte, fileInfo.Size())
	_, err = file.Read(data)
	if err != nil {
		log.Println("无法读取文件:", err)
		return
	}

	// 设置消息标志为文件传输
	var msg Msg

	// 发送消息
	msg.Write(conn, senderId, receiverId, data, FLAG_FILE)

}
