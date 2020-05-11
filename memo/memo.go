package main

import (
	"os"
	"io/ioutil"
	"log"
	"flag"
	"bytes"
	"fmt"
	"time"
)

const layout = "01/02/06, 15:04"

func main() {
	file := flag.String("file", "/u/marius@grailbio.com/Notes/Memo", "note file")

	f, err := os.OpenFile(*file, os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("read input: %v", err)
	}
	lines := bytes.Split(b, []byte{'\n'})
	if _, err := fmt.Fprintf(f, "%s:\t%s\n", time.Now().Format(layout), lines[0]); err != nil {
		log.Fatal(err)
	}
	for i, line := range lines[1:] {
		if i == len(lines)-2 && len(line) == 0{
			break
		}
		if _, err := f.Write([]byte{'\t', '\t', '\t', '\t'}); err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write(line); err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			log.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}
