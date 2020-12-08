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
)

var idDisk uint8 = 0

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
		fmt.Println("mkdisk")
		mkdisk(flagsArray[1:])
		break
	case "rmdisk":
		fmt.Println("rmkisk")
		rmdisk(flagsArray[1:])
		break
	case "fdisk":
		fmt.Println("fdisk")
		break
	case "mount":
		fmt.Println("mount")
		break
	case "unmount":
		fmt.Println("unmount")
		break
	case "rep":
		fmt.Println("rep")
		break
	case "exit":
		fmt.Println("run finisehd")
		os.Exit(1)
	}
}

func mkdisk(args []string) {

	mapArgs := getArgs(args)

	if mapArgs["path"] == "" && mapArgs["size"] == "" {
		fmt.Println("at least the path and size arguments must come")
	} else {

		mapArgs["path"] = fixPaths(mapArgs["path"])

		fit := strings.ToLower(mapArgs["fit"])
		unit := strings.ToLower(mapArgs["unit"])

		var path []string = strings.SplitAfter(mapArgs["path"], "/")

		mapArgs["path"] = strings.Join(path[:len(path)-1], "")

		mbr1 := Structs.Mbr{}
		//mbr.Mbr_date_creation = time.Now().String()
		copy(mbr1.Mbr_date_creation[:], time.Now().String())

		if fit == "" {
			mbr1.Disk_fit = 'f'
		} else if fit == "bf" || fit == "ff" || fit == "wf" {
			mbr1.Disk_fit = fit[0]
		} else {
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

		if unit == "" {
			mbr1.Mbr_size = sizeFile * 1024 * 1024
		} else if unit == "m" {
			mbr1.Mbr_size = sizeFile * 1024 * 1024
		} else if unit == "k" {
			mbr1.Mbr_size = sizeFile * 1024
		} else {
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
		mbr1.Mbr_disk_signature = idDisk + 1

		data = 1
		var mbrBuffer bytes.Buffer
		_ = binary.Write(&mbrBuffer, binary.BigEndian, &mbr1)
		err = writeBytes(file, mbrBuffer.Bytes())

		if err != nil {
			return
		}
	}
}

func rmdisk(args []string) {
	mapArgs := getArgs(args)

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
		fmt.Println("at least the path and size arguments must come")
	}
}

func getArgs(args []string) map[string]string {
	mapArgs := make(map[string]string)
	var keyValue []string

	for _, item := range args {
		fmt.Println(item)
		keyValue = strings.Split(item, "->")
		mapArgs[keyValue[0]] = keyValue[1]
	}

	return mapArgs
}

func fixPaths(path string) string {
	if path[0] == '"' {
		path = path[1 : len(path)-1]
	}
	if path[0] == '/' {
		path = path[1:]
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
