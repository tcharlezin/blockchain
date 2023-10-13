package main

import (
	"flag"
	"fmt"
	"log"
)

func init() {
	log.SetPrefix("Wallet Server: ")
}

func main() {
	port := flag.Uint("port", 8080, "TCP Port number for wallet server")
	gateway := flag.String("gateway", "http://127.0.0.1:8000", "Blockchain Gateway")
	flag.Parse()

	app := NewWalletServer(uint16(*port), *gateway)

	log.Println(fmt.Sprintf("Running the server in the port :%d", uint16(*port)))

	app.Run()
}
