package main // import "marius.ae/tools/g"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	goregexp "regexp"
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

func walk(path string, g *regexp.Grep) {
	if *vflag {
		log.Printf("walk %s %s", path, g.Regexp)
	}

	paths := []string{path}

	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		paths = filepath.SplitList(os.Getenv("GPATH"))
		for i, dir := range paths {
			paths[i] = filepath.Join(dir, path)
		}
	} else if err != nil {
		// Ignore
		return
	}
	if *vflag {
		log.Printf("walk paths %v", paths)
	}

	for _, root := range paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if *vflag {
				log.Printf("walk %s", path)
			}

			if err != nil || info == nil {
				return nil
			}

			if info.Mode().IsDir() {
				switch filepath.Base(path) {
				case ".git", ".svn", "_build", "node_modules", ".mypy_cache":
					return filepath.SkipDir
				default:
					return nil
				}
			}

			if info.Mode().IsRegular() && pathOk(path) && contentOk(path) {
				g.File(path)
			}

			return nil
		})
	}
}

func pathOk(path string) bool {
	switch filepath.Ext(path) {
	case ".lst", ".asm", ".rst", ".sym", ".rel", ".map":
		return false
	default:
		return true
	}
}

func contentOk(path string) bool {
	switch filepath.Ext(path) {
	case ".a", ".pkg":
		return false
	}

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
			walk(path, &g)
		}
	}

	if !g.Match {
		os.Exit(1)
	}
}

// from cmd/go/main.go

// matchPattern(pattern)(name) reports whether
// name matches pattern.  Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
func matchPattern(pattern string) func(name string) bool {
	re := goregexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := goregexp.MustCompile(`^` + re + `$`)
	return func(name string) bool {
		return reg.MatchString(name)
	}
}
