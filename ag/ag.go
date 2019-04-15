// Ag searches open Acme windows for a regular expression, printing
// results in the manner of grep so that they are B3-clickable inside
// of Acme.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/codesearch/regexp"

	"9fans.net/go/acme"
)

var iflag = flag.Bool("i", false, "case insensitive match")

func usage() {
	fmt.Fprintf(os.Stderr, "ag regexp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	var g regexp.Grep
	g.AddFlags()
	g.Stdout = os.Stdout
	g.Stderr = os.Stderr
	g.N = true
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	pat := "(?m)" + flag.Arg(0)
	if *iflag {
		pat = "(?i)" + pat
	}
	var err error
	g.Regexp, err = regexp.Compile(pat)
	if err != nil {
		log.Fatal(err)
	}

	infos, err := acme.Windows()
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range infos {
		w, err := acme.Open(info.ID, nil)
		if err != nil {
			log.Printf("open %d: %v", info.ID, err)
			continue
		}
		b, err := w.ReadAll("body")
		if err != nil {
			log.Printf("read %d body: %v", info.ID, err)
			continue
		}
		g.Reader(bytes.NewReader(b), info.Name)
	}
}
