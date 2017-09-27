package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/pasiukevich/inmemory"

	// for file changes monitoring
	"github.com/fsnotify/fsnotify"
)

func main() {
	serversPtr := flag.String("servers", "servers.json", "List of inmemory cluster servers.")
	connsPtr := flag.String("conns", "10", "Number of connections in the pool for one server.")
	addrPtr := flag.String("addr", "127.0.0.1:10000", "Address to listen.")
	certPtr := flag.String("cert", "server.crt", "Server certificate filepath.")
	keyPtr := flag.String("key", "server.key", "Server key filepath.")

	flag.Parse()

	servers := readServers(serversPtr)

	circle := inmemory.NewCircle()
	circle.Adjust(servers...)

	go monitorServers(serversPtr, circle)

	conns, err := strconv.Atoi(*connsPtr)

	if err != nil {
		log.Fatal(err)
	}
	pool := inmemory.NewPool(conns, newConnection, servers...)

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

	log.Println("Proxy is listening on:", *addrPtr)

	// accept and handle incoming connections
	for {
		clientConn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(clientConn, pool, circle)
	}
}

func newConnection(server string) (net.Conn, error) {
	tlsConfig := tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", server, &tlsConfig)
	if err != nil {
		log.Fatal("Connection error", err)
		return nil, err
	}

	return conn, nil
}

func handleConnection(clientConn net.Conn, pool *inmemory.Pool, circle *inmemory.Circle) {
	// read/writer to the clients connection
	rw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	defer clientConn.Close()

	log.Println("Client connected to proxy from:", clientConn.RemoteAddr())

	// serve client requests
	for {
		input, err := rw.ReadString('\n')

		switch {
		case err == io.EOF:
			log.Println("Closed connection from:", clientConn.RemoteAddr())
			return
		case err != nil:
			log.Printf("Error reading command. Got: '%v' with error: %v\n", input, err)
			return
		}

		// parse the command
		fields := strings.Fields(input)

		if len(fields) > 1 {
			key := fields[1]
			server := circle.Get(key)
			serverConn, ok := pool.Get(server.Addr)
			serverrw := bufio.NewReadWriter(bufio.NewReader(serverConn), bufio.NewWriter(serverConn))

			if !ok {
				log.Println("Couldn't get the connection to", server)
			} else {
				query := input
				log.Printf("writing to server %s %s", server.Addr, query)
				serverrw.WriteString(query)
				serverrw.Flush()
				result, err := serverrw.ReadString('\n')

				log.Printf("got result: %s", string(result))
				if err != nil {
					rw.WriteString(err.Error())
				} else {
					rw.WriteString(result)
				}

				rw.Flush()
				pool.Return(server.Addr, serverConn)
			}
		} else {
			rw.WriteString("request should have at least 2 words: command and key\n")
			rw.Flush()
		}
	}
}

func readServers(filepath *string) []*inmemory.Server {
	var servers []*inmemory.Server
	serverList, err := ioutil.ReadFile(*filepath)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(serverList, &servers)

	return servers
}

func monitorServers(serversFile *string, circle *inmemory.Circle) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = watcher.Add(*serversFile)
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	for {
		select {
		case event := <-watcher.Events:
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("server list file is modified. Updating keys distribution")
				servers := readServers(serversFile)
				circle.Adjust(servers...)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}
