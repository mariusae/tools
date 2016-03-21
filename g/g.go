package main	// import "marius.ae/tools/g"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/codesearch/regexp"
)

var iflag = flag.Bool("i", false, "case insensitive match")
var vflag = flag.Bool("v", false, "verbose")

func usage() {
	fmt.Fprintf(os.Stderr, "g: query [match..]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func match1(q, p string) bool {
	if strings.IndexAny(q, "*?[") > -1 {
		ok, _ := filepath.Match(q, p)
		return ok
	} else {
		return strings.Index(p, q) > -1
	}
}

func match(query, path string) bool {
	ps := strings.Split(path, "/")
	qs := strings.Split(query, "/")
	i := 0

	for _, q := range qs[:len(qs)-1] {
		found := false
		for !found && i < len(ps)-1 {
			found = match1(q, ps[i])
			i++
		}
		if !found {
			return false
		}
	}

	p := ps[len(ps)-1]
	q := qs[len(qs)-1]

	return match1(q, p)
}

func walk(path string, g *regexp.Grep) {
	if *vflag {
		log.Printf("walk %s %s", path, g.Regexp)
	}

	query := "*"

	if strings.Contains(path, "...") {
		i := strings.Index(path, "...")
		if i == 0 {
			query = path[3:]
			path = "."
		} else {
			path, query = path[0:i], path[i+3:]
		}
	}

	paths := []string{path}

	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		paths = filepath.SplitList(os.Getenv("GPATH"))
		for i := range paths {
			paths[i] = filepath.Join(paths[i], path)
		}
	} else if err != nil {
		// Ignore
		return
	}

	if *vflag {
		log.Printf("query %s; dirs: %s", query, strings.Join(paths, ","))
	}

	for _, path := range paths {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if *vflag {
				log.Printf("walk %s", path)
			}

			if err != nil || info == nil {
				return nil
			}

			if info.Mode().IsDir() {
				switch filepath.Base(path) {
				case ".git", ".svn", "_build":
					return filepath.SkipDir
				default:
					return nil
				}
			}

			if info.Mode().IsRegular() && match(query, path) && contentOk(path) && pathOk(path) {
				g.File(path)
			}

			return nil
		})
	}
}

func walkParents(path string, g *regexp.Grep) {

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	for {
		file := filepath.Join(dir, path)

		if *vflag {
			log.Printf("walkParents %s %s", file, g.Regexp)
		}

		if info, err := os.Stat(file); err == nil && info.Mode().IsRegular() {
			g.File(file)
		}

		if dir == "/" {
			break
		}

		dir = filepath.Dir(dir)
	}
}

func pathOk(path string) bool {
	switch filepath.Ext(path) {
	case ".lst", ".asm", ".rst", ".sym", ".rel":
		return false
	default:
		return true
	}
}

func contentOk(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var data [512]byte

	n, err := io.ReadFull(f, data[:])
	if n == 0 && (err == io.ErrUnexpectedEOF || err == io.EOF) {
		return true
	} else if n <= 0 {
		return false
	}

	// This may be too strict -- we could also implement
	// something similar to silver searcher's heuristic.
	tpe := http.DetectContentType(data[:n])

	/*
		if !strings.HasPrefix(tpe, "text/") {
			log.Printf("Bad content type \"%s\" for %s", tpe, path)
		}
	*/

	return strings.HasPrefix(tpe, "text/")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("g: ")
	var g regexp.Grep
	g.AddFlags()
	g.Stdout = os.Stdout
	g.Stderr = os.Stderr
	g.N = true
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
	}

	pat := "(?m)" + args[0]
	if *iflag {
		pat = "(?i)" + pat
	}

	re, err := regexp.Compile(pat)
	if err != nil {
		log.Fatal(err)
	}

	g.Regexp = re

	if len(args) == 1 {
		walk(".", &g)
	} else {
		for _, path := range args[1:] {
			if strings.HasPrefix(path, "~") {
				walkParents(path[1:], &g)
			} else {
				walk(path, &g)
			}
		}
	}

	if !g.Match {
		os.Exit(1)
	}
}
