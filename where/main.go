package main

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
	"9fans.net/go/plumb"
)

const wname = "+where"

func main() {
	// TODO: recreate windows when needed
	acme.AutoExit(true)

	go plumber()
	select {}
}

func plumber() {
	var win *awin

	fid, err := plumb.Open("where", 0)
	if err != nil {
		acme.Errf("", "plumb: %v", err)
		return
	}
	r := bufio.NewReader(fid)
plumber:
	for {
		var m plumb.Message
		if err := m.Recv(r); err != nil {
			acme.Errf("", "plumb recv: %v", err)
			return
		}
		if m.Dst != "where" {
			acme.Errf("", "plumb recv: unexpected dst: %s\n", m.Dst)
			continue
		}
		var file, addr, action string
		for attr := m.Attr; attr != nil; attr = attr.Next {
			switch attr.Name {
			case "file":
				file = attr.Value
			case "addr":
				addr = attr.Value
			case "action":
				action = attr.Value
			}
		}
		if win == nil {
			win = new(awin)
			win.name = "+where"
			win.open()
		}
		switch action {
		case "pop":
			win.pop()
			win.ExecGet()
			continue plumber
		}

		// Bookmarks must be unique.
		for _, b := range win.list {
			if b.file == file && b.addr == addr {
				continue plumber
			}
		}
		win.list = append(win.list, bookmark{file, addr, string(m.Data)})
		win.ExecGet()
	}
}

type bookmark struct {
	file, addr string
	context    string
}

type awin struct {
	*acme.Win
	name string
	list []bookmark
}

func (w *awin) open() {
	var err error
	w.Win, err = acme.New()
	if err != nil {
		log.Fatalf("cannot create acme window: %v", err)
	}
	w.Name(w.name)
	w.Ctl("cleartag")
	w.Fprintf("tag", " Put Pop")
	go w.EventLoop(w)
}

func (w *awin) ExecGet() {
	if err := w.execGet(); err != nil {
		w.Err(err.Error())
	}
}

func (w *awin) execGet() (err error) {
	var b strings.Builder
	for _, bookmark := range w.list {
		b.WriteString(bookmark.file)
		b.WriteString(":")
		b.WriteString(bookmark.addr)
		b.WriteString("\n")
		b.WriteString("\t")
		c := strings.TrimSpace(bookmark.context)
		c = strings.Replace(c, "\t", "    ", -1)
		c = strings.Replace(c, "\n", "\n\t", -1)
		b.WriteString(c)
		b.WriteString("\n")
	}
	w.Clear()
	w.PrintTabbed(b.String())
	w.Addr("$")
	w.Ctl("dot=addr")
	w.Ctl("clean")
	w.Ctl("show")
	return nil
}

func (w *awin) Execute(cmd string) bool {
	switch cmd {
	case "Put":
		if err := w.put(); err != nil {
			w.Errf("Put: %v", err)
			return true
		}
		w.Ctl("clean")
		return true
	case "Pop":
		w.pop()
		w.ExecGet()
		return true
	}
	return false
}

func (w *awin) Look(text string) bool {
	return false
}

func (w *awin) put() error {
	body, err := w.ReadAll("body")
	if err != nil {
		return err
	}
	var list []bookmark
	scan := bufio.NewScanner(bytes.NewReader(body))
	for scan.Scan() {
		line := scan.Text()
		if !strings.HasPrefix(line, "\t") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			// TODO: maybe we should not have an opinion
			// about the file/address here.
			var (
				parts = strings.SplitN(line, ":", 2)
				file  = parts[0]
				addr  string
			)
			if len(parts) == 2 {
				addr = parts[1]
			}
			list = append(list, bookmark{file: file, addr: addr})
		} else if len(list) > 0 {
			b := &list[len(list)-1]
			var buf strings.Builder
			buf.WriteString((*b).context)
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(strings.TrimPrefix(line, "\t"))
			(*b).context = buf.String()
		} // otherwise, ignore spurious line
	}
	if err := scan.Err(); err != nil {
		return err
	}
	w.list = list
	return nil
}

func (w *awin) pop() {
	if len(w.list) == 0 {
		w.Err("no bookmarks")
		return
	}
	b := w.list[len(w.list)-1]
	w.list = w.list[:len(w.list)-1]
	edit(b.file, b.addr)
}

func edit(path, line string) {
	if line != "" {
		path += ":" + line
	}

	out, err := exec.Command("plumb", "-d", "edit", path).CombinedOutput()
	if err != nil {
		log.Fatalf("plumb: %v\n%s", err, out)
	}
}
