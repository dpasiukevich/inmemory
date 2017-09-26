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

	// fill data store
	testData(client, 200)

	// check size
	client.Exec("size", []string{})

	if client.reply != "200" {
		t.Error("Didn't create dummy data")
		t.Errorf("Expected reply: \"200\", got: \"%s\"", client.reply)
	}
	if client.err != nil {
		t.Error("Didn't create dummy data")
		t.Errorf("Expected reply: <nil>, got: %#v", client.err)
	}

	// change TTL of only one item
	client.Exec("TTL", []string{"key50", "4"})

	time.Sleep(7 * time.Second)

	client.Exec("size", []string{})

	// check that size decreased by one
	if client.reply != "199" {
		t.Error("Didn't remove the expired key")
		t.Errorf("Expected reply: \"199\", got: \"%s\"", client.reply)
	}
	if client.err != nil {
		t.Error("Didn't remove the expired key")
		t.Errorf("Expected reply: <nil>, got: %#v", client.err)
	}
}

func TestExec(t *testing.T) {
	client := setupTestClient()

	// check initial client state
	if client.cmd != "" {
		t.Errorf("Expected cmd: \"\", got: \"%s\"", client.cmd)
	}

	client.Exec("set", []string{"x", "42"})

	// check client state after command execution
	if client.cmd != "SET" {
		t.Errorf("Expected cmd: \"SET\", got: \"%s\"", client.cmd)
	}

	// check nonexistent command
	client.Exec("WRONGCOMMAND", []string{})

	// client store previous successful request state
	if client.cmd != "SET" {
		t.Errorf("Expected cmd: \"SET\", got: \"%s\"", client.cmd)
	}
}
