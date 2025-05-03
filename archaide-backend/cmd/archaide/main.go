package main

import (
	"flag"

	"github.com/Driemtax/Archaide/internal/server"
)

var addr = flag.String("addr", ":3030", "http service address")

func main() {
	flag.Parse()
	server.Run(addr)
}
