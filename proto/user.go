package proto

type User struct {
	Id     [10]byte
	Passwd [20]byte
}

var User1 = User{
	Id:     [10]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
	Passwd: [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
}

var User2 = User{
	Id:     [10]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	Passwd: [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
}
