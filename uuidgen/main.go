package main

import (
	"flag"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func main() {

	columns := -1
	lines := -1
	flag.IntVar(&columns, "l", 1, "Linhas")
	flag.IntVar(&lines, "c", 1, "Colunas")
	flag.Parse()
	for l := 0; l < lines; l++ {
		for c := 0; c < columns; c++ {
			if c > 0 {
				fmt.Print(";")
			}
			fmt.Printf("%s", uuids())
		}
		fmt.Print("\n")
	}
}

func uuids() string {
	u1 := uuid.NewV4()
	return u1.String()
}
