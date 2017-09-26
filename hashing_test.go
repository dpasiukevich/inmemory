package inmemory

import (
	"strconv"
	"testing"
)

func TestCircleAddServer(t *testing.T) {
	circle := NewCircle()

	server := &Server{"127.0.0.1:3000", 50}

	circle.AddServer(server)

	if _, ok := circle.servers[server]; !ok {
		t.Error("Expected to add server to circle", *server)
	}
}

func TestCircleRemoveServer(t *testing.T) {
	circle := NewCircle()
	server := &Server{"127.0.0.1:3000", 50}

	circle.AddServer(server)
	circle.AddServer(&Server{"127.0.0.1:3001", 50})

	circle.RemoveServer(server)

	if _, ok := circle.servers[server]; ok {
		t.Error("Expected to remove server from circle", *server)
	}

	circle = NewCircle()
	server = &Server{"127.0.0.1:3000", 50}
	circle.AddServer(server)
	circle.RemoveServer(server)

	if _, ok := circle.servers[server]; ok {
		t.Error("Expected to remove server from circle", *server)
	}

	circle = NewCircle()
	server = &Server{"127.0.0.1:3000", 50}
	circle.RemoveServer(server)
	if len(circle.servers) > 0 {
		t.Error("Expected manage deletion of non-included server")
	}
}

func TestCircleGet(t *testing.T) {
	circle := NewCircle()
	server := &Server{"127.0.0.1:3000", 1}

	circle.AddServer(&Server{"127.0.0.1:3001", 0})
	circle.AddServer(&Server{"127.0.0.1:3002", 0})
	circle.AddServer(server)

	resultServer := circle.Get("TEST KEY")

	if *resultServer != *server {
		t.Errorf("Expected to locate key on server %v, got: %v\n", *server, *resultServer)
	}
}

func TestNewCircle(t *testing.T) {
	circle := NewCircle()
	if circle == nil {
		t.Errorf("Expected to create hash circle")
	}
}

func TestCircleAdjust(t *testing.T) {

	serversNum := 50
	weight := 200
	var servers []*Server

	for i := 0; i < serversNum; i++ {
		s := &Server{"127.0.0.1:" + strconv.Itoa(8080+i), weight}
		servers = append(servers, s)
	}

	circle := NewCircle()
	circle.Adjust(servers...)

	for _, server := range servers {
		if _, ok := circle.servers[server]; !ok {
			t.Errorf("Expected to have %v server in circle\n", server)
		}
	}

	failedServer := servers[13]
	servers = append(servers[:13], servers[14:]...)
	circle.Adjust(servers...)

	if _, ok := circle.servers[failedServer]; ok {
		t.Errorf("Expected to delete last server %v from circle\n", failedServer)
	}
}
