package Structs

import (
	"time"
)

type Mbr struct {
	Mbr_size           uint64
	Mbr_date_creation  time.Time
	Mbr_disk_signature uint8
	Disk_fit           byte
	Mbr_partition      [4]Partition
}
