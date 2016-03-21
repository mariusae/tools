// Edit is a file finder/plumber for Acme.
//
// Usage:
//
// 	edit query [dirs...]
//
// Edit executes a query against a set of directories (default: .).
// If there is exactly one result, edit will automatically plumb the
// files, similar to Plan 9's B command.
//
// The EDITPATH environment variable is a colon-separated list of
// directories to look for files.
//
// Using the invocation:
//
//	edit dir:query
//
// Edit executes the query against  x/dir for every directory x in EDITPATH.
//
// Edit traverses each given directory, skipping common database paths
// (.git, .svn), and matches each entry against the query.
//
// Queries are partial paths. A query matches a candidate path
// when each path element in the query matches a path element
// in the candidate path. The elements have to appear in the same
// order, but not all path elements from the candidate path are
// required to match.
//
// A query path element matches a candidate path element if
// (1) it is a substring of the path element; or (2) it is a glob pattern
// (containing any of "*?[") that matches according to filepath.Match.
package main	// import "marius.ae/tools/edit"

//go:generate stringer -type=matchKind

// 	- Scoring/select first

/*


	- TODO: pick the first by mtime (newest wins)
	- also list by mtime...

	- TODO: when mtimes are equal--resolve to shortest(?)

	- TODO: go back to "..."?

	- TODO: list missed files when picking the first

	- TODO: exact match always wins(?)

	- TODO: allow for partial glob matches, e.g.,
		tweet*serv
	should match
		tweet_service.thrift



*/

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

var ignoreDirs = map[string]bool{
	".git": true,
	".svn": true,
	"_build": true,
}

type matchKind int

const (
	matchNone matchKind = iota
	matchPartial
	matchExact
)

type hit struct {
	path  string
	match matchKind
	info  os.FileInfo
}

type hitOrder []hit

func (h hitOrder) Len() int      { return len(h) }
func (h hitOrder) Swap(i, j int) { h[j], h[i] = h[i], h[j] }

func (h hitOrder) Less(i, j int) bool {
	return h[i].match < h[j].match || (h[i].match == h[j].match &&
		h[i].info.ModTime().Before(h[j].info.ModTime()))
}

var printOnly = flag.Bool("n", false, "Don't plumb results, just print them.")
var print1Only = flag.Bool("n1", false, "Like -n, but print only one.")
var dirsOnly = flag.Bool("d", false, "Edit directories, not files.")
var debug = flag.Bool("debug", false, "Debug output")

func (h hit) String() string {
	return fmt.Sprintf("%s %s: %s", h.info.ModTime().Format(time.RFC3339), h.match, h.path)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: edit query [dir...]\n")
	fmt.Fprint(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func match1(q, p string) matchKind {
	if strings.IndexAny(q, "*?[") > -1 {
		ok, _ := filepath.Match(q, p)
		if ok {
			return matchPartial
		} else {
			return matchNone
		}
	} else {
		if strings.Index(p, q) < 0 {
			return matchNone
		} else if len(p) == len(q) {
			return matchExact
		} else {
			return matchPartial
		}
	}
}

func match(query, path string) matchKind {
	ps := strings.Split(path, "/")
	qs := strings.Split(query, "/")
	i := 0

	for _, q := range qs[:len(qs)-1] {
		found := false
		for !found && i < len(ps)-1 {
			found = match1(q, ps[i]) != matchNone
			i++
		}
		if !found {
			return matchNone
		}
	}

	p := ps[len(ps)-1]
	q := qs[len(qs)-1]

	return match1(q, p)
}

func plumb(path string) {
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

func main() {
	log.SetFlags(0)
	log.SetPrefix("edit: ")
	flag.Usage = usage
	flag.Parse()

	if *printOnly && *print1Only {
		usage()
	}

	query := "*"
	if flag.NArg() > 0 {
		query = flag.Arg(0)
	}

	cased := false
	for _, r := range query {
		cased = cased || unicode.IsUpper(r)
	}

	if !cased {
		query = strings.ToLower(query)
	}

	var dirs []string
	if flag.NArg() > 1 {
		for i := 1; i < flag.NArg(); i++ {
			path := flag.Arg(i)
			fi, err := os.Stat(path)
			if err != nil && os.IsNotExist(err) {
				dirs = filepath.SplitList(os.Getenv("EDITPATH"))
				for i := range dirs {
					dirs[i] = filepath.Join(dirs[i], path)
				}

			} else if err != nil || !fi.Mode().IsDir() {
				continue
			}

			dirs = append(dirs, path)
		}
	} else {
		dirs = []string{"."}
	}

	if *debug {
		log.Printf("query \"%s\" dirs \"%v\"", query, strings.Join(dirs, ", "))
	}

	hits := []hit{}

	var matchMode os.FileMode
	if *dirsOnly {
		matchMode = os.ModeDir
	}

	// TODO: refactor this to be nicer
	if strings.HasPrefix(query, "~") {
		query := query[1:]
		for _, dir := range dirs {
			dir, err := filepath.Abs(dir)
			if err != nil {
				log.Fatal(err)
			}

			for {
				path := filepath.Join(dir, query)

				info, err := os.Stat(path)
				if err != nil || info == nil {
					goto Next
				}

				if (info.Mode()&os.ModeType)^matchMode != 0 {
					goto Next
				}

				if !cased {
					path = strings.ToLower(path)
				}

				if match := match(query, path); match != matchNone && (info.Mode().IsDir() || contentOk(path)) {
					hits = append(hits, hit{path, match, info})
				}

			Next:
				if dir == "/" {
					break
				}

				dir = filepath.Dir(dir)
			}
		}
	} else {
		for _, d := range dirs {
			filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					switch e := err.(type) {
					case *os.PathError:
						if e.Err.Error() == "no such file or directory" {
							return nil
						}
					default:
						return err
					}
				}

				if info == nil {
					return nil
				}

				if info.Mode().IsDir() {
					if _, ok := ignoreDirs[filepath.Base(path)]; ok {
						return filepath.SkipDir
					}
				}

				if (info.Mode()&os.ModeType)^matchMode != 0 {
					return nil
				}

				rel, err := filepath.Rel(d, path)
				if err != nil {
					return err
				}

				if !cased {
					rel = strings.ToLower(rel)
				}

				if match := match(query, rel); match != matchNone && (info.Mode().IsDir() || contentOk(path)) {
					hits = append(hits, hit{path, match, info})
				}
				return nil
			})
		}
	}

	sort.Sort(sort.Reverse(hitOrder(hits)))

	if *debug {
		for _, hit := range hits {
			fmt.Println(hit)
		}
	}

	if *printOnly {
		var prev string
		for _, hit := range hits {
			if prev != hit.path {
				fmt.Println(hit.path)
			}
			prev = hit.path
		}
	} else if *print1Only {
		if len(hits) == 0 {
			os.Exit(1)
		}
		fmt.Println(hits[0].path)
	} else if len(hits) > 0 {
		plumb(hits[0].path)
	}
}
