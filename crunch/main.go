package main

import (
	"9fans.net/go/draw"
)

var display *draw.Display

func main() {
	var err error
	display, err = draw.Init(nil, "", name, "")
	if err != nil {
		log.Fatal(err)
	}
}
