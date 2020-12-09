package Commands

import (
	"../Structs"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var idDisk int64 = 0

func CommandLine(command string) {

	var flagsArray []string
	flagsArray = strings.Split(command, " -")

	switch strings.ToLower(flagsArray[0]) {
	case "exec":
		fmt.Println("exec")
		break
	case "pause":
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Press Intro Key to continue...")
		_, _ = reader.ReadString('\n')
	case "mkdisk":
		mkdisk(flagsArray[1:])
		break
	case "rmdisk":
		rmdisk(flagsArray[1:])
		break
	case "fdisk":
		fdisk(flagsArray[1:])
		break
	case "mount":
		mount(flagsArray[1:])
		break
	case "unmount":
		unmount(flagsArray[1:])
		break
	case "rep":
		rep(flagsArray[1:])
		break
	case "exit":
		fmt.Println("run finished")
		os.Exit(1)
	}
}

func mkdisk(args []string) {

	mapFlags := map[string]bool{
		"path": true,
		"size": true,
		"fit":  true,
		"unit": true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("at least the path and size arguments must come")
		return
	}
	if mapArgs == nil {
		return
	}

	if mapArgs["path"] == "" && mapArgs["size"] == "" {
		fmt.Println("at least the path and size arguments must come")
		return
	} else {

		mapArgs["path"] = fixPaths(mapArgs["path"])

		fit := strings.ToLower(mapArgs["fit"])
		unit := strings.ToLower(mapArgs["unit"])

		var path = strings.SplitAfter(mapArgs["path"], "/")

		mapArgs["path"] = strings.Join(path[:len(path)-1], "")

		mbr1 := Structs.Mbr{}
		//mbr.Mbr_date_creation = time.Now().String()
		copy(mbr1.Mbr_date_creation[:], time.Now().String())

		if !validFit(fit, &mbr1) {
			fmt.Println(fit + " unsupported value")
			return
		}

		sizeFile, err := strconv.ParseInt(mapArgs["size"], 10, 64)

		if err != nil {
			log.Fatal("Size must be a number", err)
			return
		}

		if sizeFile <= 0 {
			fmt.Println("Size must be greater than 0")
			return
		}

		if !validUnit(unit, sizeFile, &mbr1) {
			fmt.Println(unit + " unsupported value")
			return
		}

		//Create the Directory
		err = os.MkdirAll(mapArgs["path"], 0777)
		if err != nil {
			log.Fatal("Error creating path: ", err)
			return
		}

		//Create the File
		file, err := os.Create(mapArgs["path"] + path[len(path)-1])
		defer file.Close()
		if err != nil {
			log.Fatal("Error creating file: ", err)
			return
		}

		//write 0 to the beginning of the file
		_, _ = file.Seek(0, 0)

		var data int8 = 0

		var startBuffer bytes.Buffer
		_ = binary.Write(&startBuffer, binary.BigEndian, &data)
		err = writeBytes(file, startBuffer.Bytes())

		if err != nil {
			return
		}

		//write 0 to the end of the file
		_, _ = file.Seek(mbr1.Mbr_size-1, 0)

		var endBuffer bytes.Buffer
		_ = binary.Write(&endBuffer, binary.BigEndian, &data)
		err = writeBytes(file, endBuffer.Bytes())

		if err != nil {
			return
		}

		//Write MBR
		_, _ = file.Seek(0, 0)
		idDisk++
		mbr1.Mbr_disk_signature = idDisk

		var mbrBuffer bytes.Buffer
		_ = binary.Write(&mbrBuffer, binary.BigEndian, &mbr1)
		err = writeBytes(file, mbrBuffer.Bytes())

		if err != nil {
			return
		}
	}
}

func rmdisk(args []string) {
	mapFlags := map[string]bool{
		"path": true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("at least the path argument must come")
		return
	}
	if mapArgs == nil {
		return
	}

	if mapArgs["path"] != "" {

		if fileExists(mapArgs["path"]) {

			var option string
			fmt.Print("Are you sure you want to delete the file [Y/n]: ")
			_, _ = fmt.Scanf("%s\n", &option)

			if strings.ToLower(option) == "y" {
				err := os.Remove(mapArgs["path"])
				if err != nil {
					log.Fatal("The file could not be deleted", err)
					return
				}
			}

		} else {
			fmt.Println("File or directory doesn't exist")
			return
		}

	} else {
		fmt.Println("at least the path argument must come")
	}
}

func fdisk(args []string) {

	mapFlags := map[string]bool{
		"path":   true,
		"size":   true,
		"fit":    true,
		"unit":   true,
		"type":   true,
		"name":   true,
		"add":    true,
		"delete": true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("there must be arguments")
		return
	}
	if mapArgs == nil {
		return
	}

	size := mapArgs["size"]
	unit := mapArgs["unit"]
	typeP := mapArgs["type"]
	fit := mapArgs["fit"]
	deletePart := mapArgs["delete"]
	add := mapArgs["add"]
	name := mapArgs["name"]
	path := mapArgs["path"]

	if mapArgs["path"] != "" && name != "" {
		path = fixPaths(path)

		if fileExists(mapArgs["path"]) {

			file, err := os.OpenFile(path, os.O_RDWR, 0777)
			defer file.Close()
			if err != nil {
				log.Fatal("The file could not be opened", err)
				return
			}

			mbr := Structs.Mbr{}
			mbrSize := int64(unsafe.Sizeof(mbr))
			_, _ = file.Seek(0, 0)
			mbr = retriveMbr(file, mbrSize, mbr)

			if size != "" && deletePart == "" && add == "" {
				//Create Partition

				if flag, indexPartition := noPartitions(mbr.Mbr_partition); flag {
					sizePartition, err := strconv.ParseInt(size, 10, 64)
					if err != nil {
						log.Fatal("size could not be converted to int", err)
						return
					}

					if !validUnitPartition(unit, sizePartition, &mbr.Mbr_partition[0]) {
						fmt.Println(unit + " unsupported value")
						return
					}

					difference := mbr.Mbr_size - (mbrSize + mbr.Mbr_partition[0].Part_size)

					if difference > 0 {

						if !validFitPartition(fit, &mbr.Mbr_partition[0]) {
							fmt.Println(fit + " unsupported value")
							return
						}

						if !validType(typeP, &mbr.Mbr_partition[0]) {
							fmt.Println(typeP + " unsupported value")
							return
						}

						mbr.Mbr_partition[0].Part_status = 1
						mbr.Mbr_partition[0].Part_start = mbrSize
						copy(mbr.Mbr_partition[0].Part_name[:], name)

						_, _ = file.Seek(0, 0)

						var mbrBuffer bytes.Buffer
						_ = binary.Write(&mbrBuffer, binary.BigEndian, &mbr)
						_ = writeBytes(file, mbrBuffer.Bytes())

					}

				} else {
					if mbrSize == mbr.Mbr_partition[indexPartition].Part_start {

					}
				}

			} else if add != "" && deletePart == "" && size == "" {
				//Increment Size of partition
			} else if deletePart != "" && size == "" && add == "" {
				//Delete Partition
			} else if size == "" && deletePart == "" && add == "" {
				fmt.Println("at least the size argument must come")
				return
			}

		} else {
			fmt.Println("File or directory doesn't exist")
			return
		}
	} else {
		fmt.Println("at least the path and name arguments must come")
	}

}

func mount(args []string) {
	mapFlags := map[string]bool{
		"path": true,
		"name": true,
	}
	mapArgs, count := getArgs(args, mapFlags)

	if count != 0 {
		fmt.Println("There are more arguments than supported")
		return
	}
	if len(args) == 0 {
		fmt.Println("at least the path and name arguments must come")
		return
	}
	if mapArgs == nil {
		return
	}

	if mapArgs["paht"] != "" && mapArgs["name"] != "" {

	} else {
		fmt.Println("at least the path and name arguments must come")
	}
}

func unmount(args []string) {
	mapFlags := map[string]bool{
		"id": true,
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

	if mapArgs["id"] != "" {

	} else {
		fmt.Println("at least the id argument must come")
	}
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

	if mapArgs["id"] != "" && mapArgs["name"] != "" && mapArgs["path"] != "" {

	} else {
		fmt.Println("at least the path, name and id arguments must come")
	}
}

func retriveMbr(file *os.File, size int64, mbr Structs.Mbr) Structs.Mbr {
	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &mbr)

	if err != nil {
		log.Fatal(err)
	}

	return mbr
}

func getArgs(args []string, flags map[string]bool) (map[string]string, int) {
	mapArgs := make(map[string]string)
	var keyValue []string
	i := 0
	sizeFlags := len(flags)
	for _, item := range args {
		if i < sizeFlags {
			i++
			keyValue = strings.Split(item, "->")
			if flags[strings.ToLower(keyValue[0])] {
				mapArgs[strings.ToLower(keyValue[0])] = strings.ToLower(keyValue[1])
			} else {
				fmt.Println("The argument" + keyValue[0] + "is not accepted")
				return nil, 0
			}
		} else {
			return nil, i
		}
	}

	return mapArgs, 0
}

func fixPaths(path string) string {
	if path[0] == '"' && path[len(path)-1] == '"' {
		path = path[1 : len(path)-1]
	} else if path[0] == '"' {
		path = path[1:]
	} else if path[len(path)-1] == '"' {
		path = path[:len(path)-1]
	}
	return path
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func writeBytes(file *os.File, bytes []byte) error {
	_, err := file.Write(bytes)

	if err != nil {
		log.Fatal("Error writing to file", err)
	}
	return err
}

func readBytes(file *os.File, size int64) []byte {
	arrayBytes := make([]byte, size)

	_, err := file.Read(arrayBytes)

	if err != nil {
		log.Fatal(err)
	}

	return arrayBytes
}

func validUnit(unit string, sizeFile int64, mbr1 *Structs.Mbr) bool {
	if unit == "" {
		mbr1.Mbr_size = sizeFile * 1024 * 1024
		return true
	} else if unit == "m" {
		mbr1.Mbr_size = sizeFile * 1024 * 1024
		return true
	} else if unit == "k" {
		mbr1.Mbr_size = sizeFile * 1024
		return true
	}
	return false
}

func validUnitPartition(unit string, sizePartition int64, partition *Structs.Partition) bool {
	if unit == "" {
		partition.Part_size = sizePartition * 1024
		return true
	} else if unit == "b" {
		partition.Part_size = sizePartition
	} else if unit == "m" {
		partition.Part_size = sizePartition * 1024 * 1024
		return true
	} else if unit == "k" {
		partition.Part_size = sizePartition * 1024
		return true
	}
	return false
}

func validFit(fit string, mbr1 *Structs.Mbr) bool {
	if fit == "" {
		mbr1.Disk_fit = 'f'
		return true
	} else if fit == "bf" || fit == "ff" || fit == "wf" {
		mbr1.Disk_fit = fit[0]
		return true
	}
	return false
}

func validFitPartition(fit string, partition *Structs.Partition) bool {
	if fit == "" {
		partition.Part_fit = 'f'
		return true
	} else if fit == "bf" || fit == "ff" || fit == "wf" {
		partition.Part_fit = fit[0]
		return true
	}
	return false
}

func validType(typeP string, partition *Structs.Partition) bool {
	if typeP == "" {
		partition.Part_type = 'p'
		return true
	} else if typeP == "p" || typeP == "e" || typeP == "l" {
		partition.Part_type = typeP[0]
		return true
	}

	return false
}

func noPartitions(partitions [4]Structs.Partition) (bool, int) {
	for i := 0; i < 4; i++ {
		if partitions[i].Part_status != 0 {
			return false, i
		}
	}

	return true, -1
}

func partitionsCreated(partitions [4]Structs.Partition) int {
	var count = 0
	for i := 0; i < 4; i++ {
		if partitions[i].Part_status != 0 {
			count++
		}
	}

	return count
}

func sortPartition(partitions [4]Structs.Partition) [4]Structs.Partition {

	if partitionsCreated(partitions) == 1 {
		return partitions
	}
	var aux Structs.Partition

	for i := 0; i < 3; i++ {
		for j := 1; j < 4; j++ {
			if partitions[i].Part_status > partitions[j].Part_status {
				aux = partitions[i]
				partitions[i] = partitions[j]
				partitions[j] = aux
			}
		}
	}

	return partitions
}
