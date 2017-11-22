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
	fmt.Println("Start gostub server...")
	fmt.Printf("port: %v, output: %v\n", *portOption, *outputPathOption)
	Run(*portOption, *outputPathOption)
}
