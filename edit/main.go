package main // import "marius.ae/tools/edit"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	"path/filepath"
)

func main() {
	log.SetPrefix("")
	log.SetFlags(0)
	listFlag := flag.Bool("n", false, "list results")
	flag.Usage = func() {
		io.WriteString(os.Stderr, `usage:
	edit query
	edit query paths...
`)
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
	}
	query := flag.Arg(0)
	var paths []string
	if flag.NArg() == 1 {
		paths = []string{"."}
	} else {
		for i := 1; i < flag.NArg(); i++ {
			arg := flag.Arg(i)
			// If it's an explicit path then use it directly, otherwise
			// find a matching EDITPATH directory.
			if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, ".") {
				paths = append(paths, arg)
				continue
			}
			for _, prefix := range filepath.SplitList(os.Getenv("EDITPATH")) {
				dirpath := filepath.Join(prefix, arg)
				info, err := os.Stat(dirpath)
				if err == nil && info.IsDir() {
					paths = append(paths, dirpath)
				}
			}
		}
	}
	if len(paths) == 0 {
		log.Fatal("no search paths found")
	}
	matches := make(map[string]bool)
	for _, root := range paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			relpath, err := filepath.Rel(root, path)
			if err != nil {
				log.Fatal(err)
			}

			dir, file := filepath.Split(relpath)
			dir = strings.TrimSuffix(dir, "/")
			if !matchf(query, file) || !contentOk(path) {
				return nil
			}

			if matches[path] {
				return nil
			}
			matches[path] = true
			return nil
		})
	}

	if len(matches) == 0 {
		os.Exit(1)
	}
	if len(matches) > 1 || *listFlag {
		sorted := make([]string, 0, len(matches))
		for filepath := range matches {
			sorted = append(sorted, filepath)
		}
		sort.Strings(sorted)
		for _, filepath := range sorted {
			fmt.Println(filepath)
		}
		os.Exit(0)
	}

	for filepath := range matches {
		plumb(filepath, "")
	}
}

func matchf(query, s string) bool {
	for _, r := range query {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i:]
	}
	return true
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

func plumb(path, line string) {
	if line != "" {
		path += ":" + line
	}

	out, err := exec.Command("plumb", "-d", "edit", path).CombinedOutput()
	if err != nil {
		log.Fatalf("plumb: %v\n%s", err, out)
	}
}
