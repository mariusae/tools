package main 

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

func main() {
	fs := flag.String("F", "\t", "field separator")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: tabulate [-F sep]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	log.SetFlags(0)
	log.SetPrefix("")
	flag.Parse()

	var tw tabwriter.Writer
	tw.Init(os.Stdout, 4, 4, 1, ' ', 0)
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		line := strings.Replace(scan.Text(), *fs, "\t", -1)
		fmt.Fprintln(&tw, line)
	}
	if err := scan.Err(); err != nil {
		log.Fatal(err)
	}
	tw.Flush()
}
