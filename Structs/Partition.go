package Structs

type Partition struct {
	Part_status byte
	Part_type   byte
	Part_fit    byte
	Part_start  int
	Part_size   int
	Part_name   [16]byte
}
