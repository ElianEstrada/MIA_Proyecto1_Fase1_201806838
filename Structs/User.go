package Structs

type User struct {
	Uid    int64
	Gid    int64
	Usr    string
	Pwd    string
	Flag   bool
	isRoot bool
}
