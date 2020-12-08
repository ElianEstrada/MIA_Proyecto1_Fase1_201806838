package Commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var path, _ = filepath.Abs(filepath.Dir(os.Args[0]))

func Init() {
	fmt.Print("elian_estrada@elian-pc:", path, "~ ")

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	CommandLine(strings.Split(text, "\n")[0])
}
