package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"strings"

	"github.com/pasiukevich/inmemory"
)

func handleConnection(conn net.Conn, dataStore *inmemory.DataStore) {

	// read/writer to the connection
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()

	// initialize client for the data store
	client := inmemory.NewClient(dataStore)

	log.Println("Client connected from:", conn.RemoteAddr())

	// serve client requests
	for {
		input, err := rw.ReadString('\n')

		switch {
		case err == io.EOF:
			log.Println("Closed connection from:", conn.RemoteAddr())
			return
		case err != nil:
			log.Printf("Error reading command. Got: '%v' with error: %v\n", input, err)
			return
		}

		// parse the command
		fields := strings.Fields(input)

		if len(fields) > 0 {
			cmd := strings.ToUpper(fields[0])

			// execute the command with given arguments
			reply, err := client.Exec(cmd, fields[1:])

			// write the reply
			if err == nil {
				rw.WriteString(reply)
			} else {
				rw.WriteString(err.Error())
			}
			rw.WriteString("\n")

			// send the reply
			rw.Flush()
		}
	}
}

func main() {
	addrPtr := flag.String("addr", "127.0.0.1:9443", "Address to listen.")
	backupPtr := flag.String("backup", "", "Path to file with backup in gob format. Used to restore previous state of server.")
	certPtr := flag.String("cert", "server.crt", "Server certificate filepath.")
	keyPtr := flag.String("key", "server.key", "Server key filepath.")

	flag.Parse()

	// create the data store
	dataStore := inmemory.New()

	// try to restore data from file if it's given
	if *backupPtr != "" {
		dataStore.FromFile(*backupPtr)
	}

	// use the certificates to setup encrypted connections
	cer, err := tls.LoadX509KeyPair(*certPtr, *keyPtr)
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", *addrPtr, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	log.Println("Server is listening on:", *addrPtr)

	// accept and handle incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn, dataStore)
	}
}
