package Structs

type SuperBlock struct {
	S_filesystem_type   int64    //save number to identify to the file system
	S_inodes_count      int64    //save number of inodes
	S_blocks_count      int64    //save number of blocks
	S_free_blocks_count int64    //content the number of free blocks
	S_free_inodes_count int64    //content the number of free inodes
	S_mtime             [19]byte //last date the system was mounted
	S_umtime            [19]byte //last date the system was unmounted
	S_mnt_count         int64    //indicates how many times the system has been mounted
	S_magic             int64    //value to identify the file system, value = 0xEF53
	S_inode_size        int64    //size of inodes
	S_block_size        int64    //size of blocks
	S_first_ino         int64    //first free inode
	S_first_blo         int64    //first free block
	S_bm_inode_start    int64    //save the init of inode's bitmap
	S_bm_block_start    int64    //save the init of block's bitmap
	S_inode_start       int64    //save the init of inode's table
	S_block_start       int64    //save the init of block's table
}
