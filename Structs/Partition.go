package Structs

type Partition struct {
	Part_status int8
	Part_type   byte
	Part_fit    byte
	Part_start  int64
	Part_size   int64
	Part_name   [16]byte
}
