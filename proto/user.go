package proto

type User struct {
	Id     uint32
	Passwd string
}

var User1 = User{
	Id:     1,
	Passwd: "1",
}

var User2 = User{
	Id:     2,
	Passwd: "2",
}

var user_db = []User{User1, User2}

var Server = User{
	Id: 0,
}

func Login(u User) (bool, string) {
	for _, user := range user_db {
		if user.Id == u.Id {
			if user.Passwd == u.Passwd {
				return true, ""
			} else {
				return false, "密码错误"
			}
		}
	}
	return false, "未找到用户"
}
