package main // import "marius.ae/tools/trim"

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetPrefix("trim: ")
	log.SetFlags(0)

	s := bufio.NewScanner(bufio.NewReader(os.Stdin))
	for s.Scan() {
		fmt.Println(strings.TrimSpace(s.Text()))
	}

	if s.Err() != nil {
		log.Fatal(s.Err())
	}
}
