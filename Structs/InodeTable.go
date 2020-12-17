package Structs

type InodeTable struct {
	I_uid   int64     //UID the user property
	I_gid   int64     //UID the group property
	I_size  int64     //size of file in bytes
	I_atime [19]byte  //last date the inode was read without modifying it
	I_ctime [19]byte  //date of create
	I_mtime [19]byte  //last date the inode was modify
	I_block [15]int64 //array blocks
	I_type  byte      //indicate if it is file or directory
	I_perm  int64     //save the permission of file or directory
}
