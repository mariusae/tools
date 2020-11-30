package main	

import (
	"path/filepath"
	"time"
	"flag"
	"log"
	"strings"
	"fmt"
	"strconv"
	"os"
	"os/exec"
	"io"
	"io/ioutil"
	"net/http"
	"bytes"

	"upspin.io/upspin"
	"upspin.io/client"
	"upspin.io/config"
	"upspin.io/transports"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("upsnarf: ")
	vflag := flag.Bool("v", false, "copy from pasteboard (macOS)")
	flag.Parse()
	if flag.NArg() > 0 && *vflag {
		log.Fatal("-v and file arguments given")
	}

	home := config.Home()
	cfg, err := config.FromFile(filepath.Join(home, "upspin", "config"))
	if err != nil {
		log.Fatal(err)
	}
	transports.Init(cfg)
	upc := client.New(cfg)
	now := time.Now()
	prefix := "marius@me.com/Snarfs/"+now.Format("2006-01-02")+"-"

	var readers []io.Reader
	if *vflag {
		readers = append(readers, bytes.NewReader(pbpaste()))
	} else if flag.NArg() > 0 {
		for i := 0; i < flag.NArg(); i++ {
			file, err := os.Open(flag.Arg(i))
			if err != nil {
				log.Fatal(err)
			}
			readers = append(readers, file)
			defer file.Close()
		}
	} else {
		readers = append(readers, os.Stdin)
	}

	for _, reader := range readers {
		files, err := upc.Glob(prefix+"*")
		if err != nil {
			log.Fatal(err)
		}
		var index int64
		for _, file := range files {
			suffix := strings.TrimPrefix(string(file.Name), prefix)
			if ext := filepath.Ext(suffix)  ;ext != "" {
				suffix = suffix[:len(suffix)-len(ext)]
			}
			i, err := strconv.ParseInt(suffix, 10, 64)
			if err != nil {
				log.Printf("not a snarf path: %s: %v", file.Name, err)
				continue
			}
			if i >= index {
				index = i+1
			}
		}

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}

		var suffix string
		switch http.DetectContentType(data) {
		case "application/octet-stream":
			suffix = ".bin"
		case "application/pdf":
			suffix = ".pdf"
		case "image/gif":
			suffix = ".gif"
		case "image/bmp":
			suffix = ".bmp"
		case "image/jpeg":
			suffix = ".jpeg"
		case "image/png":
			suffix = ".png"
		case "application/zip":
			suffix = ".zip"
		case "text/html; charset=utf-8":
			suffix = ".html"
		}

		path := upspin.PathName(fmt.Sprintf("%s%02d%s", prefix, index, suffix))

		if _, err := upc.Put(path, data); err != nil {
			log.Fatal(err)
		}
		fmt.Println(path)
	}
}

func pbpaste() []byte {
	cmd := exec.Command("pbpaste")
	p, err := cmd.Output()
	if err != nil {
		log.Fatalf("pbpaste: %v", err)
	}
	return p
}

