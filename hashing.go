package inmemory

import (
	"hash/crc32"
	"sort"
	"sync"
)

// Server struct hold the info about the server. It has an address and number of
// virtual nodes to create for the server.
type Server struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

// node and nodes represent virtual node for the server
// it allows better keys distribution in the circle
type node uint32
type nodes []node

func (n nodes) Len() int           { return len(n) }
func (n nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n nodes) Less(i, j int) bool { return n[i] < n[j] }

// Circle struct hold all information about virtual nodes and servers. Also it has
// means to add/remove servers without major key shifts.
// Its methods are safe for the concurrent use.
type Circle struct {
	sync.RWMutex
	nodes       nodes
	servers     map[*Server]struct{}
	node2server map[node]*Server
}

// NewCircle returns pointer to the Circle struct object ready for usage.
func NewCircle() *Circle {
	return &Circle{
		servers:     make(map[*Server]struct{}),
		node2server: make(map[node]*Server),
	}
}

// Adjust the number of servers in the pool. It either deletes disabled servers or
// adds new ones for the keys distribution.
func (c *Circle) Adjust(servers ...*Server) {

	newServers := make(map[*Server]struct{})

	for _, server := range servers {
		newServers[server] = struct{}{}
	}

	c.Lock()
	defer c.Unlock()
	// remove servers if needed
	for server := range c.servers {
		if _, ok := newServers[server]; !ok {
			c.remove(server)
		}
	}

	// add new servers if needed
	for server := range newServers {
		if _, ok := c.servers[server]; !ok {
			c.add(server)
		}
	}

	sort.Sort(c.nodes)
}

// AddServer to the circle.
func (c *Circle) AddServer(server *Server) {
	c.Lock()
	defer c.Unlock()
	c.add(server)
	sort.Sort(c.nodes)
}

// RemoveServer from the circle.
func (c *Circle) RemoveServer(s *Server) {

	if _, ok := c.servers[s]; ok {
		c.Lock()
		defer c.Unlock()
		c.remove(s)
	}
}

// Get key from the circle. Key distribution depends on the servers weight.
func (c *Circle) Get(key string) *Server {
	c.RLock()
	defer c.RUnlock()

	// search for the virtualnode, associated with the key
	i := c.search(key)
	if i >= len(c.nodes) {
		i = 0
	}
	return c.node2server[c.nodes[i]]
}

// adding server to the circle. Each server has number of vnodes represented as hashes
// to generate several hashes from server address, append byte for each iteration
func (c *Circle) add(server *Server) {
	if _, ok := c.servers[server]; !ok {
		c.servers[server] = struct{}{}
		serverBytes := []byte(server.Addr)
		for i := 0; i < server.Weight; i++ {
			vnodeHash := node(crc32.ChecksumIEEE(serverBytes))
			c.nodes = append(c.nodes, vnodeHash)

			// assosiate virtual node hash with the server
			c.node2server[vnodeHash] = server

			// update input to generate new hash
			serverBytes = append(serverBytes, '_')
		}
	}
}

func (c *Circle) remove(s *Server) {

	// removing last server from the circle means its nullifying
	if len(c.servers) == 1 {
		c.nodes = nodes{}
		c.servers = make(map[*Server]struct{})
		c.node2server = make(map[node]*Server)
		return
	}

	// delete vnodes associated with the server
	for nodeIndex, nodeHash := range c.nodes {
		if server := c.node2server[nodeHash]; *server == *s {
			delete(c.node2server, nodeHash)
			c.nodes = append(c.nodes[:nodeIndex], c.nodes[nodeIndex+1:]...)
		}
	}
	delete(c.servers, s)
}

// search for the key in the nodes. The node returned has hash value greater the key's hash
func (c *Circle) search(key string) int {
	searchfn := func(i int) bool {
		return c.nodes[i] >= node(crc32.ChecksumIEEE([]byte(key)))
	}

	return sort.Search(len(c.nodes), searchfn)
}
