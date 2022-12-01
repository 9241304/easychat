package main

import (
	"os"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	if os.Args[1] == "server" {
		LaunchServer()
	} else if os.Args[1] == "client" {
		LaunchClient()
	} else {
		println("Unknown command")
	}
}
