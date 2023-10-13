package main

import (
	"flag"
	"fmt"
	"log"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	port := flag.Uint("port", 8000, "TCP port number for Blockchain Server")
	flag.Parse()

	app := NewBlockchainServer(uint16(*port))

	log.Println(fmt.Sprintf("Running the server in the port :%d", uint16(*port)))

	app.Run()
}
