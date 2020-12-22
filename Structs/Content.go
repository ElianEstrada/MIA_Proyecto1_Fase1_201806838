package Structs

type Content struct {
	b_name  [12]byte //name of file or directory
	b_inode int32    //Pointer to an inode associated with the file or directory
}
