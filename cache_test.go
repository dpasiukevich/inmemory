package inmemory

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	dataStore := New()
	if dataStore == nil {
		t.Fatal("Couldn't create database.")
	}
}

func TestTTLD(t *testing.T) {
	client := setupTestClient()
	go client.Ds.ttld()

	// fill data store
	testData(client, 200)

	// check size
	client.Exec("size", []string{})

	if client.Reply != "200" {
		t.Error("Didn't create dummy data")
		t.Errorf("Expected reply: \"200\", got: \"%s\"", client.Reply)
	}
	if client.Err != nil {
		t.Error("Didn't create dummy data")
		t.Errorf("Expected reply: <nil>, got: %#v", client.Err)
	}

	// change TTL of only one item
	client.Exec("TTL", []string{"key50", "4"})

	time.Sleep(5 * time.Second)

	client.Exec("size", []string{})

	// check that size decreased by one
	if client.Reply != "199" {
		t.Error("Didn't remove the expired key")
		t.Errorf("Expected reply: \"199\", got: \"%s\"", client.Reply)
	}
	if client.Err != nil {
		t.Error("Didn't remove the expired key")
		t.Errorf("Expected reply: <nil>, got: %#v", client.Err)
	}
}

func TestExec(t *testing.T) {
	client := setupTestClient()

	// check initial client state
	if client.Cmd != "" {
		t.Errorf("Expected cmd: \"\", got: \"%s\"", client.Cmd)
	}

	client.Exec("set", []string{"x", "42"})

	// check client state after command execution
	if client.Cmd != "SET" {
		t.Errorf("Expected cmd: \"SET\", got: \"%s\"", client.Cmd)
	}

	// check nonexistent command
	client.Exec("WRONGCOMMAND", []string{})

	// client store previous successful request state
	if client.Cmd != "SET" {
		t.Errorf("Expected cmd: \"SET\", got: \"%s\"", client.Cmd)
	}
}
