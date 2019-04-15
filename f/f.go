package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: f args [dir]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	flag.Usage = usage
	flag.Parse()
	searchdir := os.Getenv("HOME")
	switch flag.NArg() {
	case 1:
	case 2:
		searchdir = flag.Arg(1)
	default:
		flag.Usage()
	}
	dir, file := filepath.Split(flag.Arg(0))
	if dir != "" {
		scan, err := mdfind(searchdir, "kind:folder", filepath.Base(dir))
		if err != nil {
			log.Fatal(err)
		}
		// Filter out the directories which do not match directly.
		for scan.Scan() {
			match := scan.Text()
			if !strings.HasSuffix(match+"/", "/"+dir) {
				continue
			}
			if file == "" {
				fmt.Println(match + "/")
				continue
			}
			names, err := readDirNames(match)
			if err != nil {
				log.Fatal(err)
			}
			for _, name := range names {
				matched, err := filepath.Match(file, name)
				if err != nil {
					log.Fatal(err)
				}
				if matched {
					path := filepath.Join(match, name)
					info, err := os.Stat(path)
					if err == nil && info.IsDir() {
						path += "/"
					}
					fmt.Println(path)
				}
			}
		}
		if err := scan.Err(); err != nil {
			log.Fatal(err)
		}
	} else {
		scan, err := mdfind(searchdir, fmt.Sprintf("kMDItemDisplayName == '%s'cd'", file))
		if err != nil {
			log.Fatal(err)
		}
		for scan.Scan() {
			fmt.Println(scan.Text())
		}
		if err := scan.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

func mdfind(dir string, args ...string) (*bufio.Scanner, error) {
	args = append([]string{"-onlyin", dir}, args...)
	cmd := exec.Command("mdfind", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return bufio.NewScanner(&out), nil
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
