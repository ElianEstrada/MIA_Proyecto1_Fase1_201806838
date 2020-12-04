package Commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
		break
	case "rmdisk":
		fmt.Println("rmkisk")
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
