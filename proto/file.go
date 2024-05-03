package proto

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

type File struct {
	Filename string
	//FileType string
	Data []byte
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

	// fmt.Println(f.Data)

	f.Filename = filepath.Base(filePath) // 提取文件名
	//f.FileType = filepath.Ext(f.Filename) // 提取文件扩展名
	filenameBytes := []byte(f.Filename)

	// 将文件名、分隔符和文件内容拼接在一起
	fileData := append(filenameBytes, f.Data...)

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
	// msg.Write(conn, senderId, receiverId, fileData, FLAG_FILE)
}

func (f *File) GetName(msg Msg) string {
	// 首先，解析消息中的文件数据
	filenameLen := int(msg.Extra_info)
	filenameBytes := msg.Data[:filenameLen]
	filename := string(filenameBytes)
	return filename
}

func (f *File) Receive(msg Msg, path string, target bool) (string, error) {
	// 首先，解析消息中的文件数据
	filenameLen := int(msg.Extra_info)
	filenameBytes := msg.Data[:filenameLen]
	fileData := msg.Data[filenameLen:]

	// 确保目标路径存在
	var targetPath string
	if target {
		targetPath = fmt.Sprintf("%v", path)
	} else {
		targetPath = fmt.Sprintf("%v/%d", path, msg.Receiver)
	}
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		err := os.MkdirAll(targetPath, 0777)
		if err != nil {
			fmt.Println(err)
		}
	}

	filename := string(filenameBytes)

	// 使用原始文件名创建文件，保存在指定的目录
	fullPath := filepath.Join(targetPath, filepath.Base(filename))
	file, err := os.Create(fullPath)
	if err != nil {
		log.Println("创建文件失败:", err)
		return "", err
	}
	defer file.Close()

	// 将接收到的数据写入文件
	_, err = file.Write(fileData)
	if err != nil {
		log.Println("写入文件失败:", err)
		return fullPath, err
	}

	fmt.Println("file received:", filename, " ", msg.Sender)
	return fullPath, nil
}
