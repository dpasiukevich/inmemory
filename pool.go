package inmemory

import (
	"log"
	"net"
	"sync"
)

// ConnFactory is a function that creates new connections for the pool.
// It receives the address and returns the connection or error.
type ConnFactory func(string) (net.Conn, error)

// Pool is a struct that manages the connections. It has a factory method to create
// new connections. Also it uses sync.Map as the number of servers is stable and
// it provides simple methods for the concurrent use.
type Pool struct {
	factory ConnFactory
	servers sync.Map
}

// NewPool returns pointer to the Pool structure.
// It has the following parameters:
// number of connections in the pool for one server address;
// factory function for the creation of new connections;
// variadic number of servers to manage pool for
func NewPool(size int, f ConnFactory, servers ...*Server) *Pool {

	pool := Pool{factory: f}

	// create buffered channel for each server to store the connections
	for _, server := range servers {
		ch := make(chan net.Conn, size)
		pool.servers.Store(server.Addr, ch)
	}

	return &pool
}

// Get returns connection for the server address and boolean value
// stating the success or failure of the method.
func (p *Pool) Get(addr string) (net.Conn, bool) {
	value, ok := p.servers.Load(addr)

	if !ok {
		return nil, false
	}
	conns := value.(chan net.Conn)

	select {
	case conn := <-conns:
		return conn, true
	default:
		if p.factory == nil {
			log.Println("Factory function is not set for pool")
			return nil, false
		}
		conn, error := p.factory(addr)

		if error != nil {
			log.Println(error)
			return nil, false
		}
		return conn, true
	}
}

// Return connection object in the connection pool. The method receives
// address of the server and the actual connection object.
// It returns the bool value that provides the result of operation.
func (p *Pool) Return(addr string, conn net.Conn) bool {
	value, ok := p.servers.Load(addr)

	if !ok {
		conn.Close()
		return false
	}
	conns := value.(chan net.Conn)

	select {
	case conns <- conn:
	default:
		conn.Close()
	}
	return true
}
