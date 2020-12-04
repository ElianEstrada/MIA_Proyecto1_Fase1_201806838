package Structs

type Ebr struct {
	Part_status byte
	Part_fit    byte
	Part_start  int
	Part_size   int
	Part_name   [16]byte
	part_next   int
}
