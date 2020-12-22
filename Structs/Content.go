package Structs

type Content struct {
	B_name  [12]byte //name of file or directory
	B_inode int32    //Pointer to an inode associated with the file or directory
}
