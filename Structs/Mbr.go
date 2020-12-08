package Structs

type Mbr struct {
	Mbr_size           int64
	Mbr_date_creation  [19]byte
	Mbr_disk_signature int64
	Disk_fit           byte
	Mbr_partition      [4]Partition
}
