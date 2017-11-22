package main

import (
	"flag"
	"fmt"
)

var (
	portOption = flag.String("p", "8181", "port number")
	outputPathOption = flag.String(
		"o", "", "output path (e.g. 'tests' -> ./tests)")
)

func init() {
	flag.Parse()
}

func main() {
	fmt.Printf("-p: %v, -o: %v", *portOption, *outputPathOption)
	Run(*portOption, *outputPathOption)
}
