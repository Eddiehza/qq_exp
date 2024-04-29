package proto

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

type File struct {
	Filename string
	FileType string
	Data     []byte
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
	f.Data = make([]byte, fileInfo.Size())
	_, err = file.Read(f.Data)
	if err != nil {
		log.Println("无法读取文件:", err)
		return
	}

	f.Filename = filepath.Base(filePath)  // 提取文件名
	f.FileType = filepath.Ext(f.Filename) // 提取文件扩展名

	// 将文件名、类型和数据封装成JSON
	fileData, err := json.Marshal(f)
	if err != nil {
		log.Println("无法序列化文件数据:", err)
		return
	}

	// 设置消息标志为文件传输
	msg := Msg{
		Sender:     senderId,
		Receiver:   receiverId,
		Flags:      FLAG_FILE,
		Data:       fileData,
		Extra_info: (uint8(len(f.Filename))),
	}

	// 发送消息
	//marshaledMsg := msg.Marshal()
	msg.Write(conn, senderId, receiverId, fileData, FLAG_FILE)

}

func (f *File) Receive(msg Msg) (string, error) {
	// 首先，解析消息中的文件数据
	var receivedFile File
	err := json.Unmarshal(msg.Data, &receivedFile)
	if err != nil {
		log.Println("解析文件数据失败:", err)
		return "", err
	}

	// 使用原始文件名创建文件
	filename := receivedFile.Filename
	file, err := os.Create(filename)
	if err != nil {
		log.Println("创建文件失败:", err)
		return "", err // 返回空文件名和错误信息
	}
	defer file.Close()

	// 将接收到的数据写入文件
	_, err = file.Write(receivedFile.Data)
	if err != nil {
		log.Println("写入文件失败:", err)
		return filename, err // 返回文件名和错误信息
	}

	// 成功，打印信息并返回文件名和nil表示没有错误
	fmt.Println("文件接收并保存为:", filename)
	return filename, nil
}
