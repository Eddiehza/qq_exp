package proto

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type User struct {
	Id     uint32
	Passwd string // 存储加盐后的密码哈希值
	Salt   string // 为每个用户生成和存储一个唯一的盐值
}

// 模拟用户数据库
var user_db = []User{
	{Id: 1, Passwd: "", Salt: ""},
	{Id: 2, Passwd: "", Salt: ""},
	{Id: 3, Passwd: "", Salt: ""},
}

var Server = User{
	Id:     0, // 服务器用户ID
	Passwd: "",
	Salt:   "",
}

func init() {
	// 初始化用户数据库，设置密码和盐值
	for i := range user_db {
		user_db[i].Salt = generateSalt()
		user_db[i].Passwd = hashPassword(fmt.Sprint(i+1), user_db[i].Salt)
	}
	// 初始化服务器的盐值和密码
	Server.Salt = generateSalt()
	Server.Passwd = hashPassword("server_password", Server.Salt)
}

func generateSalt() string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 8) // 生成8位随机盐
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func hashPassword(password string, salt string) string {
	data := []byte(salt + password)
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func Login(u User) (bool, string) {
	for _, user := range user_db {
		if user.Id == u.Id {
			hashedPassword := hashPassword(u.Passwd, user.Salt) // 使用数据库中的盐值来验证密码
			if user.Passwd == hashedPassword {
				return true, ""
			} else {
				return false, "密码错误"
			}
		}
	}
	return false, "未找到用户"
}

func GetFriends(u User) []byte {
	var userId []uint32
	for _, user := range user_db {
		if user.Id != u.Id {
			userId = append(userId, user.Id)
		}
	}
	var byteSlice []byte
	for _, id := range userId {
		buf := make([]byte, 4)                 // 创建一个4字节的切片
		binary.LittleEndian.PutUint32(buf, id) // 将uint32转换为字节切片
		byteSlice = append(byteSlice, buf...)  // 将转换后的字节切片追加到byteSlice中
	}

	return byteSlice
}
