package main // import "marius.ae/tools/run"

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"9fans.net/go/acme"
)

// TODO: "Again" command to re-execute the last command
// (which we can read from the first line.. kinda.)

//var printFlag = flag.Bool("p", false, "Print full command being executed")
var labelFlag = flag.String("l", "", "Label")
var nodirFlag = flag.Bool("n", false, "Prefix title with current working directory")
var cwd = flag.String("d", "", "dir")

func usage() {
	fmt.Fprintln(os.Stderr, "usage: run command..")
	fmt.Fprintln(os.Stderr, "options:")
	//	flag.PrintDefaults()

	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("run: ")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	var dir string

	if *cwd != "" {
		dir = *cwd
	} else {
		wid, err := strconv.Atoi(os.Getenv("winid"))
		if err != nil {
			log.Fatal(err)
		}
		w, err := acme.Open(wid, nil)
		if err != nil {
			log.Fatal(err)
		}
		bytes, err := w.ReadAll("tag")
		if err != nil {
			log.Fatal(err)
		}

		f := strings.Fields(string(bytes))
		if len(f) == 0 {
			log.Fatal("bad tag")
		}

		dir = path.Dir(f[0])
	}

	//	wname := "$" + path.Clean(flag.Arg(0))
	argpath := path.Clean(flag.Arg(0))
	_, argpath = filepath.Split(argpath)
	wname := "+" + argpath
	if *labelFlag != "" {
		wname = "+" + *labelFlag
	}

	var w *acme.Win

	windows, _ := acme.Windows()
	for _, info := range windows {
		if strings.HasSuffix(info.Name, wname) {
			ww, err := acme.Open(info.ID, nil)
			if err != nil {
				log.Fatal(err)
			}
			if err != nil {
				log.Fatal(err)
			}
			ww.Addr(",")
			ww.Write("data", nil)
			w = ww
			break

		}
	}
	if w == nil {
		ww, err := acme.New()
		if err != nil {
			log.Fatal(err)
		}
		w = ww
	}

	w.Ctl("dirty")
	defer w.Ctl("clean")

	if !*nodirFlag {
		w.Name(path.Clean(dir + "/" + wname))
	} else {
		w.Name(path.Clean(wname))
	}

	//	args := strings.Join(flag.Args(), " ")
	//	w.Fprintf("body", "$ %s\n", args)

	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.Stdout = bodyWriter{w}
	cmd.Stderr = cmd.Stdout
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		w.Fprintf("body", "error: %v\n", err)
		return
	}

	w.Ctl("cleartag")
	w.Fprintf("tag", " Kill")

	done := make(chan bool)
	go func() {
		err := cmd.Wait()
		if err != nil {
			w.Fprintf("body", "\ncommand error: %v\n", err)
		}
		done <- true
	}()

	deleted := make(chan bool, 1)
	go func() {
		for e := range w.EventChan() {
			if e.C2 == 'x' || e.C2 == 'X' {
				switch string(e.Text) {
				case "Del":
					select {
					case deleted <- true:
					default:
					}
					syscall.Kill(-cmd.Process.Pid, 2)
					continue
				case "Kill":
					syscall.Kill(-cmd.Process.Pid, 2)
					continue
				}
			}
			w.WriteEvent(e)

		}
	}()

	<-done
	w.Ctl("cleartag")

	select {
	case <-deleted:
		w.Ctl("delete")
	default:
	}
}

type bodyWriter struct {
	w *acme.Win
}

func (w bodyWriter) Write(b []byte) (int, error) {
	return w.w.Write("body", b)
}
