package main

import (
	"flag"

	"github.com/kelleyk/godebian/debfile"
)

func main() {
	if err := Main(); err != nil {
		panic(err)
	}
}

func Main() error {
	flag.Parse()
	path := flag.Arg(0)

	deb, err := debfile.LoadFromFile(path)
	if err != nil {
		return err
	}

	// fmt.Printf(" * deb: %v", deb)
	_ = deb

	return nil
}
