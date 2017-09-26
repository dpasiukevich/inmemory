package inmemory

import (
	"io/ioutil"
	"log"
	"net"
	"testing"
)

var newConnection = func(server string) (net.Conn, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestNewPool(t *testing.T) {

	pool := NewPool(10, newConnection, &Server{"server1", 50}, &Server{"server2", 50})

	servers := map[string]struct{}{
		"server1": {},
		"server2": {},
	}

	for server := range servers {
		_, ok := pool.servers.Load(server)
		if !ok {
			t.Errorf("Expected %s to be in server pool", server)
		}
	}
}

func TestPoolGet(t *testing.T) {

	log.SetOutput(ioutil.Discard)

	serverAddr := "127.0.0.1:40000"
	ln, err := net.Listen("tcp", serverAddr)
	defer ln.Close()

	if err != nil {
		t.Errorf("Expected to start tcp server listener, got error: %#v", err)
	}

	go func() {
		_, err = ln.Accept()
		if err != nil {
			t.Errorf("Expected to receive connection, got error: %#v", err)
		}
	}()

	pool := NewPool(2, newConnection, &Server{serverAddr, 50})

	conn, ok := pool.Get(serverAddr)

	if !ok {
		t.Fatalf("Expected to generate new connection to %s from connection pool", serverAddr)
	}

	if conn.RemoteAddr().String() != serverAddr {
		t.Errorf("Expected to get a connection to %v, got %v", serverAddr, conn.RemoteAddr())
	}
	pool.Return(serverAddr, conn)

	_, ok = pool.Get(serverAddr)
	if !ok {
		t.Errorf("Expected to fetch new connection to %s from connection pool", serverAddr)
	}

	conn, ok = pool.Get("127.0.0.1:0")
	if ok {
		t.Errorf("Expected not to receive a connection to non-existing server, got %#v", conn)
	}

	pool = NewPool(2, nil, &Server{serverAddr, 50})
	_, ok = pool.Get(serverAddr)
	if ok {
		t.Errorf("Expected no connection when there is no factory function")
	}

	failingFactory := func(server string) (net.Conn, error) {
		conn, err := net.Dial("ttcp", server)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	pool = NewPool(2, failingFactory, &Server{serverAddr, 50})
	_, ok = pool.Get(serverAddr)
	if ok {
		t.Errorf("Expected no connection with failing connection factory")
	}
}

func TestPoolReturn(t *testing.T) {

	serverAddr := "127.0.0.1:40000"
	ln, err := net.Listen("tcp", serverAddr)

	if err != nil {
		t.Fatalf("Expected to start tcp server listener, got error: %#v", err)
	}

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				ln.Close()
				return
			default:
				_, err = ln.Accept()
				if err != nil {
					t.Errorf("Expected to receive connection, got error: %#v", err)
				}
			}
		}
	}()

	pool := NewPool(1, newConnection, &Server{serverAddr, 50})

	conn, ok := pool.Get(serverAddr)

	if !ok {
		t.Errorf("Expected to have a connection to %s in connection pool", serverAddr)
	}
	pool.Return(serverAddr, conn)

	value := []byte{}

	// try to return more connections than the limit is
	conn, err = net.Dial("tcp", serverAddr)
	pool.Return(serverAddr, conn)

	_, err = conn.Read(value)
	if err == nil {
		t.Error("Expected to have an error for read from connection")
	}

	// return connection to non-existing server
	conn, err = net.Dial("tcp", serverAddr)
	ok = pool.Return(serverAddr+"0", conn)
	if ok {
		t.Error("Expected to have an error for returning connection to non-existent server")
	}
}
