package main 

/*

./relativize /Users/marius/d/go/src /Users/marius/tmp/foo

TODO: better name
TODO: don't relativize if we have to walk up the path

*/

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func stream(in io.Reader, base string) {
	base = path.Clean(base)
	reader := bufio.NewReader(in)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		targ := path.Clean(line)
		rel, err := filepath.Rel(base, targ)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(rel)
	}
}

func main() {
	flag.Parse()

	switch flag.NArg() {
	case 0:
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		stream(os.Stdin, wd)

	case 1:
		stream(os.Stdin, flag.Args()[0])

	case 2:
		base := path.Clean(flag.Args()[0])
		targ := path.Clean(flag.Args()[1])

		rel, err := filepath.Rel(base, targ)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println(rel)
	default:
		fmt.Fprintf(os.Stderr, "usage: basepath targpath\n")
		os.Exit(1)
	}
}
