package Commands

import (
	"../Structs"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var idDisk int64 = 0
var mapMount = make(map[string]Structs.Mount)

func CommandLine(command string) {

	var flagsArray []string
	flagsArray = strings.Split(command, " -")

	switch strings.ToLower(flagsArray[0]) {
	case "exec":
		execF(flagsArray[1:])
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

func execF(args []string) {
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
		mapArgs["path"] = fixPaths(mapArgs["path"])

		bytesRead, err := ioutil.ReadFile(mapArgs["path"])

		if err != nil {
			fmt.Println("The File doesn't exist", err)
			return
		}

		fileContet := string(bytesRead)
		var comandsFile = strings.Split(fileContet, "\r\n")

		for _, item := range comandsFile {
			if item == "" {
				continue
			} else if item[0] != '#' {
				fmt.Println(item)
				CommandLine(strings.TrimSpace(item))
			} else {
				fmt.Println(item)
			}
		}

	} else {
		fmt.Println("at least the path argument must come")
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
			fmt.Println("Size must be a number", err)
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
			fmt.Println("Error creating path: ", err)
			return
		}

		//Create the File
		file, err := os.Create(mapArgs["path"] + path[len(path)-1])
		defer file.Close()
		if err != nil {
			fmt.Println("Error creating file: ", err)
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
					fmt.Println("The file could not be deleted", err)
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

	if path != "" && name != "" {
		path = fixPaths(path)

		if fileExists(path) {

			file, err := os.OpenFile(path, os.O_RDWR, 0777)
			defer file.Close()
			if err != nil {
				fmt.Println("The file could not be opened", err)
				return
			}

			mbr := Structs.Mbr{}
			mbrSize := int64(unsafe.Sizeof(mbr))
			_, _ = file.Seek(0, 0)
			mbr = retriveMbr(file, mbrSize, mbr)

			mbr.Mbr_partition = sortPartition(mbr.Mbr_partition)

			if size != "" && deletePart == "" && add == "" {
				//Create Partition

				if flag, indexPartition := noPartitions(mbr.Mbr_partition); flag {
					sizePartition, err := strconv.ParseInt(size, 10, 64)
					if err != nil {
						fmt.Println("size could not be converted to int", err)
						return
					}

					if !validUnitPartition(unit, sizePartition, &mbr.Mbr_partition[0]) {
						fmt.Println(unit + " unsupported value")
						return
					}

					if !validType(typeP, &mbr.Mbr_partition[0]) {
						fmt.Println(typeP + " unsupported value")
						return
					}

					if mbr.Mbr_partition[0].Part_type == 'l' {
						fmt.Println("An extended partition must exist to create logical")
						return
					}

					difference := mbr.Mbr_size - (mbrSize + mbr.Mbr_partition[0].Part_size)

					if difference > 0 {
						createPartition(fit, name, file, &mbr, mbrSize)
					} else {
						fmt.Println("The partition size is too large")
						return
					}

				} else {
					countPartition := partitionsCreated(mbr.Mbr_partition)
					if mbrSize == mbr.Mbr_partition[indexPartition].Part_start && countPartition == 1 {
						sizePartition, err := strconv.ParseInt(size, 10, 64)
						if err != nil {
							fmt.Println("size could not be converted to int", err)
							return
						}

						if !validUnitPartition(unit, sizePartition, &mbr.Mbr_partition[0]) {
							fmt.Println(unit + " unsupported value")
							return
						}

						if !validType(typeP, &mbr.Mbr_partition[0]) {
							fmt.Println(typeP + " unsupported value")
							return
						}

						if mbr.Mbr_partition[0].Part_type == 'l' {
							if mbr.Mbr_partition[indexPartition].Part_type == 'e' {
								//create partition logic
							} else {
								fmt.Println("An extended partition must exist to create logical")
								return
							}
						}

						occupiedSpace := mbrSize + mbr.Mbr_partition[indexPartition].Part_size
						difference := mbr.Mbr_size - (occupiedSpace + mbr.Mbr_partition[0].Part_size)

						if difference > 0 {
							createPartition(fit, name, file, &mbr, occupiedSpace)
						} else {
							fmt.Println("The partition size is too large")
							return
						}
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

	path := mapArgs["path"]
	name := mapArgs["name"]

	if path != "" && name != "" {

		if fileExists(path) {

			file, err := os.Open(path)
			defer file.Close()
			if err != nil {
				fmt.Println("The file could not be opened", err)
				return
			}

			mbr := Structs.Mbr{}
			mbrSize := int64(unsafe.Sizeof(mbr))
			_, _ = file.Seek(0, 0)
			mbr = retriveMbr(file, mbrSize, mbr)

			if searchPartition(mbr.Mbr_partition, name) {
				mountPartition := Structs.Mount{Path: mapArgs["path"], Name: mapArgs["name"], Letter: 'a', Number: 1}
				var flag bool = true
				for key, value := range mapMount {
					if value.Path != mountPartition.Path || value.Name != mountPartition.Name {
						if key != "vd"+string(mountPartition.Letter)+strconv.Itoa(mountPartition.Number) {

						} else if value.Path == mountPartition.Path {
							mountPartition.Number++
						} else {
							mountPartition.Letter++
						}
					} else {
						fmt.Println("The partition is already mounted")
						flag = false
						return
					}
				}

				if flag {
					mountPartition.Id = "vd" + string(mountPartition.Letter) + strconv.Itoa(mountPartition.Number)
					mapMount[mountPartition.Id] = mountPartition
				}

			} else {
				fmt.Println("The partition on disk doesn't exist")
				return
			}
		} else {
			fmt.Println("The disk doesn't exist")
			return
		}
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
		if mapMount[mapArgs["id"]].Path != "" {
			delete(mapMount, mapArgs["id"])
		} else {
			fmt.Println("The Id doesn't exist")
		}
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

	path := mapArgs["path"]
	name := mapArgs["name"]
	id := mapArgs["id"]

	if id != "" && name != "" && path != "" {
		path = fixPaths(path)

		if mapMount[id].Path != "" {

			if name == "mbr" || name == "disk" {
				file, err := os.Open(mapMount[id].Path)
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
				var content string
				if name == "mbr" {
					content = repMbr(mbr, file)
				} else {
					content = repDisk(mbr, file)
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

				_, err = fileDot.WriteString(content)
				if err != nil {
					fmt.Println("Error writing to file")
					return
				}

				cmd := exec.Command("dot", "-T"+extension[1], path+extension[0]+".txt", "-o", path+extension[0]+"."+extension[1])
				_, _ = cmd.Output()

			} else {
				fmt.Println("The name is invalid")
			}

		} else {
			fmt.Println("The partition is not mounted")
			return
		}

	} else {
		fmt.Println("at least the path, name and id arguments must come")
	}
}

func repMbr(mbr Structs.Mbr, file *os.File) string {
	name := strings.Split(file.Name(), "/")
	report := "digraph G{\nbgcolor = \"#313638\"\nlabel = \"" + name[len(name)-1] + "\" fontcolor = \"white\"\nlabelloc=\"t\"\nnode[fontcolor = white color = \"#007acc\" fontsize = 15]\n"
	report += "table [shape = none label = <\n<table border = \"0\" cellspacing = \"0\" style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n"
	report += "<tr>\n<td border = \"1\" sides=\"b\"> <b> Name </b> </td>\n<td border = \"1\" sides=\"b\"> <b> Value </b> </td>\n</tr>\n"
	report += "<tr>\n<td> <b> Mbr_Size </b> </td>\n<td> " + strconv.FormatInt(mbr.Mbr_size, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> <b> Mbr_date_creation </b> </td>\n<td> " + string(mbr.Mbr_date_creation[:]) + " </td>\n</tr>\n" +
		"<tr>\n<td> <b> Mbr_disk_signature </b> </td>\n<td>" + strconv.FormatInt(mbr.Mbr_disk_signature, 10) + " </td>\n</tr>\n" +
		"<tr>\n<td> <b> Disk_fit </b> </td>\n<td>" + string(mbr.Disk_fit) + " </td>\n</tr>\n"

	for _, item := range mbr.Mbr_partition {
		if item.Part_status != 0 {
			report += "<tr>\n<td colspan=\"2\" border=\"1\" sides=\"TB\"> <b> " + string(item.Part_name[:remuveNull(item.Part_name[:])]) + " </b> </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_status </b> </td>\n<td> " + strconv.Itoa(int(item.Part_status)) + " </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_type </b> </td>\n<td> " + string(item.Part_type) + " </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_fit </b> </td>\n<td> " + string(item.Part_fit) + " </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_start </b> </td>\n<td> " + strconv.FormatInt(item.Part_start, 10) + " bytes </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_size </b> </td>\n<td> " + strconv.FormatInt(item.Part_size, 10) + " bytes </td>\n</tr>\n"
		}
	}
	report += "</table>\n>]\n"

	if flag, index := noExtended(mbr.Mbr_partition); !flag {
		_, _ = file.Seek(mbr.Mbr_partition[index].Part_start, 0)
		ebr := Structs.Ebr{}
		ebrSize := int64(unsafe.Sizeof(ebr))

		ebr = retriveEbr(file, ebrSize, ebr)
		count := 0
		for ebr.Part_next != -1 {
			count++
			report += "table" + strconv.FormatInt(int64(count), 10) + " [shape = none label = <\n<table border = \"0\" cellspacing = \"0\" style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"7\">\n"
			report += "<tr>\n<td border = \"1\" sides=\"b\"> <b> Name </b> </td>\n<td border = \"1\" sides=\"b\"> <b> Value </b> </td>\n</tr>\n"
			report += "<tr>\n<td colspan=\"2\" border=\"1\" sides=\"TB\"> <b> " + string(ebr.Part_name[:remuveNull(ebr.Part_name[:])]) + " </b> </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_status </b> </td>\n<td> " + strconv.FormatInt(int64(ebr.Part_status), 10) + " </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_fit </b> </td>\n<td> " + string(ebr.Part_fit) + " </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_start </b> </td>\n<td> " + strconv.FormatInt(ebr.Part_start, 10) + " bytes </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_size </b> </td>\n<td> " + strconv.FormatInt(ebr.Part_size, 10) + " bytes </td>\n</tr>\n" +
				"<tr>\n<td> <b> Part_next </b> </td>\n<td> " + strconv.FormatInt(ebr.Part_next, 10) + " </td>\n</tr>\n</table>\n>]\n"

			_, _ = file.Seek(ebr.Part_start, 0)
			ebr = retriveEbr(file, ebrSize, ebr)
		}

		report += "}"

	} else {
		report += "}"
	}

	return report
}

func repDisk(mbr Structs.Mbr, file *os.File) string {
	mbrSize := int64(unsafe.Sizeof(mbr))
	name := strings.Split(file.Name(), "/")
	report := "digraph G{\nbgcolor = \"#313638\"\nlabel = \"" + name[len(name)-1] + "\" fontcolor = \"white\"\nlabelloc=\"t\"\nnode[fontcolor = white color = \"#007acc\" fontsize = 15]\n" +
		"table [shape = none label = <\n<table border = \"3\" cellborder = \"1\" cellspacing = \"10\" style = \"rounded\" bgcolor=\"#1a1a1a\" cellpadding = \"20\" fixedsize =\"true\">\n" +
		"<tr>\n<td rowspan= \"2\"> MBR <br/><br/></td>\n"

	for i, item := range mbr.Mbr_partition {
		if item.Part_status != 0 {
			if item.Part_type != 'e' {
				if mbrSize == item.Part_start {
					report += "<td rowspan = \"2\">" + string(item.Part_name[:remuveNull(item.Part_name[:])]) + "<br/><br/>" + strconv.FormatFloat(porcentage(mbr.Mbr_size, item.Part_size), 'f', 2, 64) + "</td>\n"
				} else if i > 0 && (mbr.Mbr_partition[i-1].Part_start+mbr.Mbr_partition[i-1].Part_size) == item.Part_start {
					report += "<td rowspan = \"2\">" + string(item.Part_name[:remuveNull(item.Part_name[:])]) + "<br/><br/>" + strconv.FormatFloat(porcentage(mbr.Mbr_size, item.Part_size), 'f', 2, 64) + "</td>\n"
				} else {
					if i == 0 {
						difference := item.Part_start - mbrSize
						report += "<td rowspan = \"2\">Free<br/><br/>" + strconv.FormatFloat(porcentage(mbr.Mbr_size, difference), 'f', 2, 64) + "</td>\n"
					} else {
						difference := item.Part_start - (mbr.Mbr_partition[i-1].Part_start + mbr.Mbr_partition[i-1].Part_size)
						report += "<td rowspan = \"2\">Free<br/><br/>" + strconv.FormatFloat(porcentage(mbr.Mbr_size, difference), 'f', 2, 64) + "</td>\n"
					}
				}
			} else {

			}
		}
		if i == 3 {
			difference := mbr.Mbr_size - (item.Part_start + item.Part_size)
			report += "<td rowspan = \"2\">Free<br/><br/>" + strconv.FormatFloat(porcentage(mbr.Mbr_size, difference), 'f', 2, 64) + "</td>\n"
		}
	}
	report += "</tr>\n</table>\n>];\n}"
	return report
}

func createPartition(fit string, name string, file *os.File, mbr *Structs.Mbr, start int64) {
	if !validFitPartition(fit, &mbr.Mbr_partition[0]) {
		fmt.Println(fit + " unsupported value")
		return
	}

	if mbr.Mbr_partition[0].Part_type == 'e' {
		if flag, _ := noExtended(mbr.Mbr_partition); flag {
			if !createExtended(file, start) {
				return
			}
		} else {
			fmt.Println("an extended partition already exists")
			return
		}
	}

	mbr.Mbr_partition[0].Part_status = 1
	mbr.Mbr_partition[0].Part_start = start
	copy(mbr.Mbr_partition[0].Part_name[:], name)

	_, _ = file.Seek(0, 0)

	var mbrBuffer bytes.Buffer
	_ = binary.Write(&mbrBuffer, binary.BigEndian, mbr)
	err := writeBytes(file, mbrBuffer.Bytes())

	if err != nil {
		fmt.Println(err)
		return
	}
}

func createExtended(file *os.File, start int64) bool {
	ebr := Structs.Ebr{Part_status: 0, Part_start: start, Part_next: -1}

	_, _ = file.Seek(start, 0)

	var ebrBuffer bytes.Buffer
	_ = binary.Write(&ebrBuffer, binary.BigEndian, &ebr)
	err := writeBytes(file, ebrBuffer.Bytes())

	if err != nil {
		fmt.Println("Could not write the ebr ", err)
		return false
	}

	return true
}

func noExtended(partitions [4]Structs.Partition) (bool, int) {
	for i, item := range partitions {
		if item.Part_status != 0 && item.Part_type == 'e' {
			return false, i
		}
	}

	return true, -1
}

func createLogic(file *os.File, start int64, mbr Structs.Mbr) bool {

	if flag, index := noExtended(mbr.Mbr_partition); !flag {

		start := mbr.Mbr_partition[index].Part_start

		_, _ = file.Seek(start, 0)

		ebr := Structs.Ebr{}
		sizeEbr := int64(unsafe.Sizeof(ebr))
		ebr = retriveEbr(file, sizeEbr, ebr)

	} else {
		fmt.Println("An extended partition must exist to create logical")
		return false
	}

	return true
}

func retriveMbr(file *os.File, size int64, mbr Structs.Mbr) Structs.Mbr {
	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &mbr)

	if err != nil {
		fmt.Println(err)
	}

	return mbr
}

func retriveEbr(file *os.File, size int64, ebr Structs.Ebr) Structs.Ebr {
	data := readBytes(file, size)
	dataBuffer := bytes.NewBuffer(data)

	err := binary.Read(dataBuffer, binary.BigEndian, &ebr)

	if err != nil {
		fmt.Println(err)
	}

	return ebr
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
		fmt.Println("Error writing to file", err)
	}
	return err
}

func readBytes(file *os.File, size int64) []byte {
	arrayBytes := make([]byte, size)

	_, err := file.Read(arrayBytes)

	if err != nil {
		fmt.Println(err)
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
	var aux Structs.Partition

	for i := 0; i < 3; i++ {
		for j := i + 1; j < 4; j++ {
			if partitions[i].Part_start > partitions[j].Part_start {
				aux = partitions[i]
				partitions[i] = partitions[j]
				partitions[j] = aux
			}
		}
	}

	return partitions
}

func searchPartition(partitions [4]Structs.Partition, name string) bool {
	var newName = make([]byte, 16)
	copy(newName[:], name)
	for _, item := range partitions {
		if string(item.Part_name[:]) == string(newName[:]) {
			return true
		}
	}

	return false
}

func remuveNull(param []byte) int {
	for i, item := range param {
		if item == 0 {
			return i
		}
	}
	return len(param)
}

func porcentage(sizeDisk int64, size int64) float64 {
	var space float64
	space = float64((size * 100.00) / sizeDisk)

	return space
}
