package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func latinToUTF8(buffer []byte) string {
	buf := make([]rune, len(buffer))
	for i, b := range buffer {
		buf[i] = rune(b)
	}
	//fmt.Println(string(buf))
	return string(buf)
}

func logError(err error) {
	msg := fmt.Sprintf(color.RedString("ERRO: %v\n", err.Error()))
	log(msg)
}

func log(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
}
