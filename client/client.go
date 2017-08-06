package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {

	addrPtr := flag.String("addr", "127.0.0.1:9443", "Address to listen/connect.")
	flag.Parse()

	conf := &tls.Config{
		// added for the self-signed certificate
		InsecureSkipVerify: true,
	}

	// connect to the server
	conn, err := tls.Dial("tcp", *addrPtr, conf)
	if err != nil {
		log.Fatal("Connection error", err)
		return
	}

	defer conn.Close()

	// reader/writer to the connection
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// reader for the user's input
	userInput := bufio.NewReader(os.Stdin)

	// client event loop
	for {
		fmt.Print(conn.RemoteAddr(), ">")
		command, err := userInput.ReadString('\n')

		if command == "exit\n" {
			fmt.Println("Bye.")
			return
		}

		// write command to the connection's buffer
		rw.WriteString(command)

		// send the data
		rw.Flush()

		// listen for the response
		response, err := rw.ReadString('\n')
		if err != nil {
			log.Println("Error reading command. Got: '", err)
			return
		}

		fmt.Print(response)
	}
}
