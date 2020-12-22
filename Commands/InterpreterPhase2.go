package Commands

import (
	"../Structs"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

func mkfs(args []string) {

	mapFlags := map[string]bool{
		"id":   true,
		"type": true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("at least the id argument must come")
		return
	}
	if mapArgs == nil {
		return
	}

	typeFormat := mapArgs["type"]
	id := mapArgs["id"]

	if id != "" {
		if MapMount[id].Id != "" {

			file, err := os.OpenFile(MapMount[id].Path, os.O_RDWR, 0777)
			defer file.Close()
			if err != nil {
				fmt.Println("The file doesn't exist")
				return
			}

			mbr := Structs.Mbr{}

			mbr = retriveMbr(file, int64(unsafe.Sizeof(mbr)), mbr)
			mbr.Mbr_partition = sortPartition(mbr.Mbr_partition)

			var newName = make([]byte, 16)
			index := -1
			copy(newName[:], MapMount[id].Name)
			for i, item := range mbr.Mbr_partition {
				if string(item.Part_name[:]) == string(newName[:]) {
					index = i
				}
			}

			var partitionStart int64
			var sizePartition int64

			if index != -1 {
				partitionStart = mbr.Mbr_partition[index].Part_start
				sizePartition = mbr.Mbr_partition[index].Part_size

				superBlock := Structs.SuperBlock{S_filesystem_type: 3, S_magic: 61267}
				sizeJournaling := int64(unsafe.Sizeof(Structs.Journaling{}))
				sizeInodes := int64(unsafe.Sizeof(Structs.InodeTable{}))
				sizeBlocks := int64(unsafe.Sizeof(Structs.FolderBlock{}))
				sizeSuperBlock := int64(unsafe.Sizeof(superBlock))
				n := (sizePartition - sizeSuperBlock) / (sizeJournaling + 4 + sizeInodes + 3*sizeBlocks)

				copy(superBlock.S_mtime[:], time.Now().String())
				superBlock.S_mnt_count = 1
				superBlock.S_inodes_count = n
				superBlock.S_blocks_count = 3 * n
				superBlock.S_free_blocks_count = 3*n - 2
				superBlock.S_free_inodes_count = n - 2
				superBlock.S_inode_size = sizeInodes
				superBlock.S_block_size = sizeBlocks
				superBlock.S_bm_inode_start = partitionStart + sizeSuperBlock + n*sizeJournaling
				superBlock.S_bm_block_start = superBlock.S_bm_inode_start + n
				superBlock.S_inode_start = superBlock.S_bm_block_start + 3*n
				superBlock.S_block_start = superBlock.S_inode_start + n*sizeInodes
				superBlock.S_first_ino = 2
				superBlock.S_first_blo = 2

				_, _ = file.Seek(partitionStart, 0)
				var bufferSuper bytes.Buffer
				_ = binary.Write(&bufferSuper, binary.BigEndian, &superBlock)
				err = writeBytes(file, bufferSuper.Bytes())

				if err != nil {
					fmt.Println("Error writing superBlock")
					return
				}

				if typeFormat == "" || typeFormat == "full" {
					fullFormat(file, superBlock)
				} else if typeFormat == "fast" {
					fastFormat(file, superBlock)
				} else {
					fmt.Println("this value is invalid for type")
					return
				}
			} else {
				fmt.Println("partitions doesn't exists")
			}
		} else {
			fmt.Println("The Id: " + id + " doesn't exist")
			return
		}
	} else {
		fmt.Println("at least the id argument must come")
		return
	}

}

func fastFormat(file *os.File, superBlock Structs.SuperBlock) {

}

func fullFormat(file *os.File, superBlock Structs.SuperBlock) {

}

func login(args []string) {
	mapFlags := map[string]bool{
		"id":  true,
		"usr": true,
		"pwd": true,
	}
	mapArgs := make(map[string]string)
	var keyValue []string

	sizeFlags := len(mapFlags)
	for i, item := range args {
		if i < sizeFlags {
			keyValue = strings.Split(item, "->")
			if mapFlags[strings.ToLower(keyValue[0])] {
				mapArgs[strings.ToLower(keyValue[0])] = keyValue[1]
			} else {
				fmt.Println("The argument" + keyValue[0] + "is not accepted")
				return
			}
		} else {
			return
		}
	}

}
