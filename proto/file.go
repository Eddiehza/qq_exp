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

	// 确保目标路径存在
	targetPath := fmt.Sprintf("/files/%d", msg.Receiver)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		log.Println("创建目标目录失败:", err)
		return "", err
	}

	// 使用原始文件名创建文件，保存在指定的目录
	fullPath := filepath.Join(targetPath, filepath.Base(receivedFile.Filename))
	file, err := os.Create(fullPath)
	if err != nil {
		log.Println("创建文件失败:", err)
		return "", err
	}
	defer file.Close()

	// 将接收到的数据写入文件
	_, err = file.Write(receivedFile.Data)
	if err != nil {
		log.Println("写入文件失败:", err)
		return fullPath, err
	}

	fmt.Println("文件接收并保存为:", fullPath)
	return fullPath, nil
}

func (f File) ServerSend(user_id uint32, conn net.Conn) {
	dirPath := fmt.Sprintf("/files/%d", user_id)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		//fmt.Printf("服务器无用户待接收文件 %d: %v", user_id, err)
		return
	}

	for _, file := range files {
		filePath := filepath.Join(dirPath, file.Name())
		if _, err := os.Stat(filePath); err == nil {
			f.Send(conn, Server.Id, user_id, filePath)
		}
	}

}

func (f *File) ClientReceive(msg Msg) (string, error) {
	// 首先，解析消息中的文件数据
	var receivedFile File
	err := json.Unmarshal(msg.Data, &receivedFile)
	if err != nil {
		log.Println("解析文件数据失败:", err)
		return "", err
	}

	// 确保目标路径存在
	targetPath := "/clientfile"
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		log.Println("创建目标目录失败:", err)
		return "", err
	}

	// 使用原始文件名创建文件，保存在指定的目录
	fullPath := filepath.Join(targetPath, filepath.Base(receivedFile.Filename))
	file, err := os.Create(fullPath)
	if err != nil {
		log.Println("创建文件失败:", err)
		return "", err
	}
	defer file.Close()

	// 将接收到的数据写入文件
	_, err = file.Write(receivedFile.Data)
	if err != nil {
		log.Println("写入文件失败:", err)
		return fullPath, err
	}

	fmt.Println("文件接收并保存为:", fullPath)
	return fullPath, nil
}
