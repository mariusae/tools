package main 

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// last element
// edit server/foo
// edit foo   .. if not found in the current directory, begin search path.
// edit foo/bar  .. search path too (fuzzy find on component).  first found...
// edit blah foo/bar

var printOnly = flag.Bool("n", false, "Don't plumb results, just print them.")
var onlyOne = flag.Bool("1", false, "Exit after the first file found.")
var dirsOnly = flag.Bool("d", false, "Edit directories, not files.")

func main() {
	log.SetPrefix("edit: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
	}

	var matchMode os.FileMode
	if *dirsOnly {
		matchMode = os.ModeDir
	}

	var dirmatch, dirprefix, filematch func(string) bool

	if *dirsOnly {
		dirprefix, dirmatch = matchPattern(flag.Arg(0))
		filematch = func(string) bool { return true }
	} else {
		dirpat, filepat := filepath.Split(flag.Arg(0))
		dirpat = strings.TrimSuffix(dirpat, "/")
		//log.Printf("dirpat %v filepat %v", dirpat, filepat)
		dirprefix, dirmatch = matchPattern(dirpat)
		filematch = func(file string) bool {
			match, err := filepath.Match(filepat, file)
			if err != nil {
				log.Fatal(err)
			}
			return match
		}
	}
	var paths []string
	switch flag.NArg() {
	case 1:
		paths = filepath.SplitList(os.Getenv("EDITPATH"))
	default:
		paths = flag.Args()[1:]
	}

	line := ""

	dirprefix = dirprefix

	seen := make(map[string]bool)
	for _, root := range paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			relpath, err := filepath.Rel(root, path)
			if err != nil {
				log.Fatal(err)
			}
			if info.IsDir() && !dirprefix(relpath) {
				return filepath.SkipDir
			}

			matched := false

			if *dirsOnly {
				matched = dirmatch(relpath)
			} else {
				dir, file := filepath.Split(relpath)
				dir = strings.TrimSuffix(dir, "/")
				matched = dirmatch(dir) && filematch(file) && contentOk(path)
			}
			matched = matched && (info.Mode()&os.ModeType)^matchMode == 0

			if !matched {
				return nil
			}

			if seen[path] {
				return nil
			}
			seen[path] = true

			if *printOnly {
				fmt.Printf("%s\n", path)
			} else {
				plumb(path, line)
			}

			if *onlyOne {
				return errors.New("done") // XXX
			}
			return nil
		})
	}
	/*
	   Outer:
	   	for _, root := range paths {
	   		w := NewWalker(root)
	   		for w.Next() {
	   			if  {
	   				continue
	   			}

	   		}
	   	}
	*/
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: edit ...\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func plumb(path, line string) {
	if line != "" {
		path += ":" + line
	}

	out, err := exec.Command("plumb", "-d", "edit", path).CombinedOutput()
	if err != nil {
		log.Fatalf("plumb: %v\n%s", err, out)
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

// from cmd/go/main.go

// matchPattern(pattern)(name) reports whether
// name matches pattern.  Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
func matchPattern(pattern string) (prefix, match func(name string) bool) {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := regexp.MustCompile(`^` + re + `$`)
	pfx, _ := reg.LiteralPrefix()
	//	log.Printf("prefix %v match %v", pfx, reg)
	prefix = func(name string) bool {
		return name == "." || strings.HasPrefix(name, pfx) || strings.HasPrefix(pfx, name)
	}
	match = func(name string) bool {
		return reg.MatchString(name)
	}
	return
}
