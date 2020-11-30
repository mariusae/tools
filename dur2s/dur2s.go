package main 

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		d, err := time.ParseDuration(s.Text())
		if err != nil {
			log.Fatalf("parsing %q: %v", s.Text(), err)
		}
		fmt.Println(d.Seconds())
	}
}
