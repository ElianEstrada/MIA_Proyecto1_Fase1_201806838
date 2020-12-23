package Commands

import (
	"../Structs"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var sesion Structs.User

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
				sizeBlockFile := int64(unsafe.Sizeof(Structs.FileBlock{}))
				sizeBlockPointer := int64(unsafe.Sizeof(Structs.PointerBlock{}))
				fmt.Println(sizeBlockFile, sizeBlockPointer)
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
				superBlock.S_first_ino = 1
				superBlock.S_first_blo = 1

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
	_, _ = file.Seek(superBlock.S_bm_inode_start, 0)
	format := make([]byte, superBlock.S_inodes_count+superBlock.S_blocks_count)
	var bufferFormat bytes.Buffer
	_ = binary.Write(&bufferFormat, binary.BigEndian, &format)
	err := writeBytes(file, bufferFormat.Bytes())

	if err != nil {
		fmt.Println("Error fast Format")
		return
	}

	var data byte
	data = 1
	for i := 0; i < 2; i++ {
		var bufferBitmap bytes.Buffer
		_, _ = file.Seek(superBlock.S_bm_inode_start+int64(i), 0)
		_ = binary.Write(&bufferBitmap, binary.BigEndian, &data)
		_ = writeBytes(file, bufferBitmap.Bytes())
	}

	for i := 0; i < 2; i++ {
		var bufferBitmap bytes.Buffer
		_, _ = file.Seek(superBlock.S_bm_block_start+int64(i), 0)
		_ = binary.Write(&bufferBitmap, binary.BigEndian, &data)
		_ = writeBytes(file, bufferBitmap.Bytes())
	}

	inode := Structs.InodeTable{
		I_uid:  1,
		I_gid:  1,
		I_type: '0',
		I_perm: 444,
	}

	copy(inode.I_ctime[:], time.Now().String())
	for i, _ := range inode.I_block {
		inode.I_block[i] = -1
	}

	inode.I_block[0] = 0

	_, _ = file.Seek(superBlock.S_inode_start, 0)
	var bufferInodes bytes.Buffer
	_ = binary.Write(&bufferInodes, binary.BigEndian, &inode)
	err = writeBytes(file, bufferInodes.Bytes())

	if err != nil {
		fmt.Println("Error writing inode")
	}

	inode.I_type = '1'
	copy(inode.I_ctime[:], time.Now().String())
	inode.I_block[0] = 1

	_, _ = file.Seek(superBlock.S_inode_start+superBlock.S_inode_size, 0)
	var bufferInode2 bytes.Buffer
	_ = binary.Write(&bufferInode2, binary.BigEndian, &inode)
	err = writeBytes(file, bufferInode2.Bytes())

	if err != nil {
		fmt.Println("Error writing inode")
	}

	folderBlock := Structs.FolderBlock{}
	folderBlock.B_content[0] = Structs.Content{B_inode: 0}
	copy(folderBlock.B_content[0].B_name[:], ".")
	folderBlock.B_content[1] = Structs.Content{B_inode: 0}
	copy(folderBlock.B_content[1].B_name[:], "..")
	folderBlock.B_content[2] = Structs.Content{B_inode: 1}
	copy(folderBlock.B_content[2].B_name[:], "user.txt")
	folderBlock.B_content[3].B_inode = -1
	folderBlock.B_typeBlock = '0'
	fileBlock := Structs.FileBlock{}
	copy(fileBlock.B_content[:], "1, G, root \n1, U, root, 123 \n")
	fileBlock.B_typeBlock = '1'

	_, _ = file.Seek(superBlock.S_block_start, 0)

	var bufferFolder bytes.Buffer
	_ = binary.Write(&bufferFolder, binary.BigEndian, &folderBlock)
	err = writeBytes(file, bufferFolder.Bytes())
	if err != nil {
		fmt.Println("Error writing folder Block")
		return
	}

	_, _ = file.Seek(superBlock.S_block_start+superBlock.S_block_size, 0)

	var bufferFile bytes.Buffer
	_ = binary.Write(&bufferFile, binary.BigEndian, &fileBlock)
	err = writeBytes(file, bufferFile.Bytes())
	if err != nil {
		fmt.Println("Error writing file Block")
		return
	}
}

func fullFormat(file *os.File, superBlock Structs.SuperBlock) {
	_, _ = file.Seek(superBlock.S_inode_start, 0)
	format := make([]byte, superBlock.S_inodes_count*superBlock.S_inode_size+superBlock.S_blocks_count*superBlock.S_block_size)
	var bufferFormat bytes.Buffer
	_ = binary.Write(&bufferFormat, binary.BigEndian, &format)
	err := writeBytes(file, bufferFormat.Bytes())

	if err != nil {
		fmt.Println("Error full format")
		return
	}

	fastFormat(file, superBlock)
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

func logout() {
	if sesion.Flag {
		sesion.Flag = false
		return
	}

	fmt.Println("There is no active session")
}

func rep(args []string) {
	mapFlags := map[string]bool{
		"path": true,
		"name": true,
		"id":   true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("at least the path, name and id arguments must come")
		return
	}
	if mapArgs == nil {
		return
	}

	path := mapArgs["path"]
	name := mapArgs["name"]
	id := mapArgs["id"]

	if id != "" && name != "" && path != "" {
		path = fixPaths(path)

		if MapMount[id].Path != "" {
			if validateName(name) {

				file, err := os.Open(MapMount[id].Path)
				defer file.Close()
				if err != nil {
					fmt.Println("Error trying to open file ", err)
					return
				}

				mbr := Structs.Mbr{}
				mbrSize := int64(unsafe.Sizeof(mbr))
				_, _ = file.Seek(0, 0)
				mbr = retriveMbr(file, mbrSize, mbr)

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

				if index != -1 {
					partitionStart = mbr.Mbr_partition[index].Part_start

					_, _ = file.Seek(partitionStart, 0)
					superBlock := Structs.SuperBlock{}
					superBlock = retriveSuperBlock(file, int64(unsafe.Sizeof(superBlock)), superBlock)

					var dot string

					switch name {
					case "inode":
						dot = inode(file, superBlock)
						break
					case "journaling":
						break
					case "block":
						dot = block(file, superBlock)
						break
					case "bm_inode":
						dot = bmInode(file, superBlock)
						break
					case "bm_block":
						dot = bmBlock(file, superBlock)
						break
					case "tree":
						break
					case "sb":
						dot = sb(superBlock)
						break
					case "file":
						break
					}

					nameFile := strings.SplitAfter(path, "/")
					extension := strings.Split(nameFile[len(nameFile)-1], ".")
					path := strings.Join(nameFile[:len(nameFile)-1], "")

					err = os.MkdirAll(path, 0777)
					if err != nil {
						fmt.Println(err)
						return
					}

					fileDot, err := os.Create(path + extension[0] + ".txt")
					defer fileDot.Close()
					if err != nil {
						fmt.Println("The file could not be created", err)
						return
					}

					_, err = fileDot.WriteString(dot)
					if err != nil {
						fmt.Println("Error writing to file")
						return
					}

					if name != "bm_inode" && name != "bm_block" {
						cmd := exec.Command("dot", "-Tsvg", path+extension[0]+".txt", "-o", path+extension[0]+".svg")
						_, _ = cmd.Output()
					}

				}

			} else {
				fmt.Println("The name is invalid")
				return
			}

		} else {
			fmt.Println("The partition is not mounted")
			return
		}

	} else {
		fmt.Println("at least the path, name and id arguments must come")
	}
}

func sb(superBlock Structs.SuperBlock) string {
	report := "digraph G{\nrankdir = \"LR\"\nnodeSep = \"2\"\nbgcolor = \"#313638\"\nlabel = \"Super Block\" fontcolor = \"white\"\nlabelloc=\"t\"\n" +
		"node[shape = none fontcolor = white color = \"#007acc\" fontsize = 15]\nedge[color = white]\ntable8 [label = <\n    <table border = \"0\" cellspacing = \"0\" " +
		"style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n" +
		"<tr>\n<td border = \"1\" sides = \"b\"> Name </td>\n<td border = \"1\" sides = \"b\"> Value </td>\n</tr>\n" +
		"<tr>\n<td> s_inodes_count </td>\n<td> " + strconv.FormatInt(superBlock.S_inodes_count, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_blocks_count </td>\n<td> " + strconv.FormatInt(superBlock.S_blocks_count, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_free_inodes_count </td>\n<td> " + strconv.FormatInt(superBlock.S_free_inodes_count, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_free_blocks_count </td>\n<td> " + strconv.FormatInt(superBlock.S_free_blocks_count, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_mtime </td>\n<td> " + string(superBlock.S_mtime[:]) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_umtime </td>\n<td> " + string(superBlock.S_umtime[:remuveNull(superBlock.S_umtime[:])]) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_mnt_count </td>\n<td> " + strconv.FormatInt(superBlock.S_mnt_count, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_magic </td>\n<td> " + strconv.FormatInt(superBlock.S_magic, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_inode_size </td>\n<td> " + strconv.FormatInt(superBlock.S_inode_size, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_block_size </td>\n<td> " + strconv.FormatInt(superBlock.S_block_size, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_first_ino </td>\n<td> " + strconv.FormatInt(superBlock.S_first_ino, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_first_blo </td>\n<td> " + strconv.FormatInt(superBlock.S_first_blo, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_bm_inode_start </td>\n<td> " + strconv.FormatInt(superBlock.S_bm_inode_start, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_bm_block_start </td>\n<td> " + strconv.FormatInt(superBlock.S_bm_block_start, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_inode_start </td>\n<td> " + strconv.FormatInt(superBlock.S_inode_start, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> s_block_start </td>\n<td> " + strconv.FormatInt(superBlock.S_block_start, 10) + " </td>\n</tr>\n" +
		"</table>\n>]\n}"

	return report
}

func inode(file *os.File, superBlock Structs.SuperBlock) string {
	report := "digraph G{\nnodeSep = \"2\"\nbgcolor = \"#313638\"\nlabel = \"Inodes\" fontcolor = \"white\"\nlabelloc=\"t\"\n" +
		"node[shape = none fontcolor = white color = \"#007acc\" fontsize = 15]\nedge[color = white]\n"

	_, _ = file.Seek(superBlock.S_bm_inode_start, 0)
	bitmapInodes := retriveBitMap(file, superBlock.S_inodes_count)
	inodes := Structs.InodeTable{}
	count := -1
	for i, item := range bitmapInodes {
		if item != 0 {
			count++
			_, _ = file.Seek(superBlock.S_inode_start+int64(i)*superBlock.S_inode_size, 0)
			inodes = retriveInodes(file, superBlock.S_inode_size, inodes)
			report += "table" + strconv.FormatInt(int64(i), 10) + " [label = <\n    <table border = \"0\" cellspacing = \"0\" " +
				"style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n" +
				"<tr>\n<td border = \"1\" sides = \"b\" colspan=\"2\"> Inode " + strconv.FormatInt(int64(i), 10) + " </td>\n</tr>\n" +
				"<tr>\n<td> i_uid </td>\n<td> " + strconv.FormatInt(inodes.I_uid, 10) + " </td>\n</tr>\n" +
				"<tr>\n<td> i_gid </td>\n<td> " + strconv.FormatInt(inodes.I_gid, 10) + " </td>\n</tr>\n" +
				"<tr>\n<td> i_size </td>\n<td> " + strconv.FormatInt(inodes.I_size, 10) + " </td>\n</tr>\n" +
				"<tr>\n<td> i_ctime </td>\n<td> " + string(inodes.I_ctime[:]) + " </td>\n</tr>\n"
			for j := 0; j < 15; j++ {
				if inodes.I_block[j] != -1 {
					report += "<tr>\n<td> i_block" + strconv.FormatInt(int64(j), 10) + " </td>\n<td> " + strconv.FormatInt(inodes.I_block[j], 10) + " </td>\n</tr>\n"
				}
			}
			if inodes.I_type == 0 {
				report += "<tr>\n<td> i_type </td>\n<td> 0 </td>\n</tr>\n"
			} else {
				report += "<tr>\n<td> i_type </td>\n<td> 1 </td>\n</tr>\n"
			}
			report += "<tr>\n<td> i_perm </td>\n<td> " + strconv.FormatInt(inodes.I_perm, 10) + " </td>\n</tr>\n" +
				"</table>\n>]\n"
		}
		if int64(count) == superBlock.S_first_ino {
			break
		}
	}
	report += "}"
	return report
}

func block(file *os.File, superBlock Structs.SuperBlock) string {
	report := "digraph G{\nnodeSep = \"2\"\nbgcolor = \"#313638\"\nlabel = \"Blocks\" fontcolor = \"white\"\nlabelloc=\"t\"\n" +
		"node[shape = none fontcolor = white color = \"#007acc\" fontsize = 15]\nedge[color = white]\n"

	_, _ = file.Seek(superBlock.S_bm_block_start, 0)
	bitmapBlocks := retriveBitMap(file, superBlock.S_blocks_count)

	folderBlock := Structs.FolderBlock{}
	fileBlock := Structs.FileBlock{}
	pointerBlock := Structs.PointerBlock{}

	for i, item := range bitmapBlocks {
		if item != 0 {
			_, _ = file.Seek(superBlock.S_block_start+int64(i)*superBlock.S_block_size, 0)
			typeB := typeBlock(file, superBlock.S_block_size)

			if typeB == 0 {
				_, _ = file.Seek(superBlock.S_block_start+int64(i)*superBlock.S_block_size, 0)
				folderBlock = retriveFolderBlock(file, superBlock.S_block_size, folderBlock)
				report += "table" + strconv.FormatInt(int64(i), 10) + " [label = <\n    <table border = \"0\" cellspacing = \"0\" " +
					"style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n" +
					"<tr>\n<td border = \"1\" sides = \"b\" colspan=\"2\"> FolderBlock " + strconv.FormatInt(int64(i), 10) + " </td>\n</tr>\n"
				for j := 0; j < 4; j++ {
					if folderBlock.B_content[j].B_inode != -1 {
						report += "<tr>\n<td>" + string(folderBlock.B_content[j].B_name[:remuveNull(folderBlock.B_content[j].B_name[:])]) + " </td>\n<td> " + strconv.FormatInt(int64(folderBlock.B_content[j].B_inode), 10) + " </td>\n</tr>\n"
					}
				}
				report += "</table>\n>]\n"
			} else if typeB == 1 {
				_, _ = file.Seek(superBlock.S_block_start+int64(i)*superBlock.S_block_size, 0)
				fileBlock = retriveFileBlock(file, superBlock.S_block_size, fileBlock)
				report += "table" + strconv.FormatInt(int64(i), 10) + " [label = <\n    <table border = \"0\" cellspacing = \"0\" " +
					"style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n" +
					"<tr>\n<td border = \"1\" sides = \"b\"> FileBlock " + strconv.FormatInt(int64(i), 10) + " </td>\n</tr>\n" +
					"<tr>\n<td>" + string(fileBlock.B_content[:remuveNull(fileBlock.B_content[:])]) + " </td>\n</tr>\n" +
					"</table>\n>]\n"
			} else {
				_, _ = file.Seek(superBlock.S_block_start+int64(i)*superBlock.S_block_size, 0)
				pointerBlock = retrivePointerBlock(file, superBlock.S_block_size, pointerBlock)
				report += "table" + strconv.FormatInt(int64(i), 10) + " [label = <\n    <table border = \"0\" cellspacing = \"0\" " +
					"style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n" +
					"<tr>\n<td border = \"1\" sides = \"b\"> PointerBlock " + strconv.FormatInt(int64(i), 10) + " </td>\n</tr>\n" +
					"<tr>\n<td>" + string(fileBlock.B_content[:remuveNull(fileBlock.B_content[:])]) + " </td>\n</tr>\n" +
					"</table>\n>]\n"
			}
		}
		if int64(i) == superBlock.S_first_ino {
			break
		}
	}
	report += "}"
	return report
}

func typeBlock(file *os.File, size int64) int {
	folderBlock := Structs.FolderBlock{}
	folderBlock = retriveFolderBlock(file, size, folderBlock)

	if folderBlock.B_typeBlock == '0' {
		return 0
	} else if folderBlock.B_typeBlock == '1' {
		return 1
	} else {
		return 2
	}
}

func bmInode(file *os.File, superBlock Structs.SuperBlock) string {
	_, _ = file.Seek(superBlock.S_bm_inode_start, 0)
	bitmapInodes := retriveBitMap(file, superBlock.S_inodes_count)

	count := 1
	var bitmap string
	for _, item := range bitmapInodes {
		if item == 0 {
			bitmap += "0\t"
		} else {
			bitmap += "1\t"
		}
		if count == 20 {
			bitmap += "\n"
			count = 0
		}

		count++
	}

	return bitmap
}

func bmBlock(file *os.File, superBlock Structs.SuperBlock) string {
	_, _ = file.Seek(superBlock.S_bm_block_start, 0)
	bitmapInodes := retriveBitMap(file, superBlock.S_blocks_count)

	count := 1
	var bitmap string
	for _, item := range bitmapInodes {
		if item == 0 {
			bitmap += "0\t"
		} else {
			bitmap += "1\t"
		}
		if count == 20 {
			bitmap += "\n"
			count = 0
		}

		count++
	}

	return bitmap
}

func retriveSuperBlock(file *os.File, size int64, superBlock Structs.SuperBlock) Structs.SuperBlock {
	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &superBlock)

	if err != nil {
		fmt.Println(err)
	}

	return superBlock
}

func retriveBitMap(file *os.File, size int64) []byte {
	array := make([]byte, size)
	data := readBytes(file, size)

	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &array)

	if err != nil {
		fmt.Println(err)
	}

	return array
}

func retriveFileBlock(file *os.File, size int64, fileBlock Structs.FileBlock) Structs.FileBlock {

	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &fileBlock)

	if err != nil {
		fmt.Println(err)
	}

	return fileBlock
}

func retriveFolderBlock(file *os.File, size int64, folderBlock Structs.FolderBlock) Structs.FolderBlock {

	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &folderBlock)

	if err != nil {
		fmt.Println(err)
	}

	return folderBlock
}

func retrivePointerBlock(file *os.File, size int64, pointerBlock Structs.PointerBlock) Structs.PointerBlock {

	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &pointerBlock)

	if err != nil {
		fmt.Println(err)
	}

	return pointerBlock
}

func retriveInodes(file *os.File, size int64, inodes Structs.InodeTable) Structs.InodeTable {
	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &inodes)

	if err != nil {
		fmt.Println(err)
	}

	return inodes
}

func validateName(name string) bool {
	nameValid := []string{"inode", "block", "journaling", "bm_inode", "bm_block", "tree", "sb", "file"}

	for _, item := range nameValid {
		if item == name {
			return true
		}
	}

	return false
}
