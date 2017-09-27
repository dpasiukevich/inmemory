package inmemory

import (
	"io/ioutil"
	"log"
	"strconv"
	"testing"
)

var (
	// general test cases for each command
	cases = map[string][]struct {
		// name of the test
		name string
		// list of arguments for the command
		args          []string
		expectedReply string
		expectedError error
	}{
		"SET": {
			{"valid string", []string{"test_key", "test_value"}, "OK", nil},
			{"reset value", []string{"test_key", "test_value"}, "OK", nil},
			{"with TTL value", []string{"key", "value", "15"}, "OK", nil},
			{"reset TTL", []string{"test_key", "test_value", "25"}, "OK", nil},
			{"empty key", []string{"", "empty_key"}, "OK", nil},
			{"empty value", []string{"empty_value", ""}, "OK", nil},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"1 argument", []string{"key"}, "", errArgumentNumber},
			{"4 arguments", []string{"key", "value", "42", "huh?"}, "", errArgumentNumber},
			{"wrong TTL format", []string{"key", "value", "fifteen"}, "", errTTLFormat},
			{"TTL less than 0", []string{"key", "value", "-42"}, "", errTTLValue},
		},
		"GET": {
			{"existing value", []string{"x"}, "15", nil},
			{"get same value again", []string{"x"}, "15", nil},
			{"nonexistent value", []string{"y"}, "", errNoItem},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"2 arguments", []string{"item1", "item2"}, "", errArgumentNumber},
		},
		"SIZE": {
			{"correct usage", []string{}, "10", nil},
			{"1 argument", []string{"x"}, "", errArgumentNumber},
		},
		"REMOVE": {
			{"correct usage", []string{"key2"}, "OK", nil},
			{"delete same key again", []string{"key2"}, "", errNoItem},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"2 arguments", []string{"x", "y"}, "", errArgumentNumber},
		},
		"KEYS": {
			{"1 argument", []string{"x"}, "", errArgumentNumber},
		},
		"TTL": {
			{"correct usage", []string{"key0", "25"}, "OK", nil},
			{"set TTL on missing key", []string{"x", "25"}, "OK", nil},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"1 argument", []string{"x"}, "", errArgumentNumber},
			{"ttl not a number", []string{"x", "y"}, "", errTTLFormat},
			{"ttl is less than 0", []string{"x", "-1"}, "", errTTLValue},
		},
		"LSET": {
			{"correct usage", []string{"list", "2", "10"}, "OK", nil},
			{"wrong arguments number", []string{"list", "2"}, "", errArgumentNumber},
			{"wrong index format", []string{"list", "index", "10"}, "", errIndexFormat},
			{"index out of range", []string{"list", "10", "10"}, "", errIndexRange},
			{"set not to list", []string{"x", "0", "0"}, "", errNotList},
		},
		"LPUSH": {
			{"correct usage", []string{"list", "value"}, "OK", nil},
			{"push to the same", []string{"list", "value1"}, "OK", nil},
			{"try to push to string", []string{"x", "value"}, "", errNotList},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"1 argument", []string{}, "", errArgumentNumber},
			{"4 arguments", []string{}, "", errArgumentNumber},
		},
		"LGET": {
			{"correct usage", []string{"list", "0"}, "value", nil},
			{"get outside of range", []string{"list", "99"}, "", errIndexRange},
			{"get not from list", []string{"x", "0"}, "", errNotList},
			{"wrong index format", []string{"list", "index"}, "", errIndexFormat},
			{"get from unexisting list", []string{"list1", "0"}, "", errNoItem},
			{"0 arguments", []string{}, "", errArgumentNumber},
			{"1 argument", []string{}, "", errArgumentNumber},
			{"3 arguments", []string{"list", "3", "something"}, "", errArgumentNumber},
		},
		"HSET": {
			{"correct usage", []string{"hash", "x", "value"}, "OK", nil},
			{"wrong arguments number", []string{"hash", "key"}, "", errArgumentNumber},
			{"insert in the same map", []string{"hash", "y", "value"}, "OK", nil},
			{"set on existing object", []string{"x", "key", "value"}, "", errNotHash},
		},
		"HGET": {
			{"correct usage", []string{"hash", "key"}, "value", nil},
			{"wrong arguments number", []string{"hash", "key", "value"}, "", errArgumentNumber},
			{"get from not hash", []string{"x", "key"}, "", errNotHash},
			{"get nonexistent item", []string{"hash", "key1"}, "", errNoKeyHash},
			{"get from nonexistent hash", []string{"hash1", "key"}, "", errNoItem},
		},
	}
)

// setup new data store and create client object for it
func setupTestClient() *Client {
	dataStore := New()

	client := &Client{
		ds:    dataStore,
		cmd:   "",
		reply: "",
	}

	return client
}

// generation of simply dummy data for testing
func testData(client *Client, n int) {
	for i := 0; i < n; i++ {
		value := strconv.Itoa(i)
		client.Exec("set", []string{"key" + value, value})
	}
}

// runner method takes the command string
// and runs the tests for the test cases
// checks are done for the expected and real Err and Reply output
func runner(t *testing.T, command string, client *Client) {

	// find the test cases for the command
	cases, ok := cases[command]

	if !ok {
		t.Fatal("No tests cases for", command)
	}

	// run tests for the test cases
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			// set the client fields, preparing to run the command
			client.cmd = command
			client.args = tc.args
			client.reply = ""
			client.err = nil

			// get the actual command function by name
			cmd, ok := commands[command]

			if !ok {
				t.Fatal("No such command:", command)
			}

			// run the command for testing
			cmd(client)

			// check output reply, real and expected
			if client.reply != tc.expectedReply {
				t.Errorf("Expected reply: \"%s\", got: \"%s\"", tc.expectedReply, client.reply)
			}

			// check errors, real and expected
			if client.err != tc.expectedError {
				t.Errorf("Expected error: %#v, got: %#v", tc.expectedError, client.err)
			}
		})
	}
}

func TestSet(t *testing.T) {
	runner(t, "SET", setupTestClient())
}

func TestGet(t *testing.T) {
	client := setupTestClient()

	_, err := client.Exec("set", []string{"x", "15"})

	if err != nil {
		t.Fatalf("Couldn't set up test value (set x 15), got: %v", err)
	}

	runner(t, "GET", client)
}

func TestSize(t *testing.T) {

	empty := setupTestClient()

	reply, err := empty.Exec("size", []string{})

	if reply != "0" {
		t.Errorf("Wrong size of empty cache. Expected reply: \"%s\", got:\"%s\"", "0", reply)
	}
	if err != nil {
		t.Errorf("Wrong size of empty cache. Expected error: %#v, got: %#v", err, nil)
	}

	filled := setupTestClient()

	testData(filled, 10)

	runner(t, "SIZE", filled)
}

func TestRemove(t *testing.T) {
	client := setupTestClient()

	testData(client, 10)

	runner(t, "REMOVE", client)
}

func TestKeys(t *testing.T) {
	client := setupTestClient()

	testData(client, 4)

	runner(t, "KEYS", client)

	client.Exec("KEYS", []string{})

	if client.err != nil {
		t.Errorf("Get all keys. Expected error: %#v, got: %#v", nil, client.err)
	}
}

func TestTTL(t *testing.T) {
	client := setupTestClient()

	testData(client, 4)

	runner(t, "TTL", client)

	log.SetOutput(ioutil.Discard)
	// should handle wrong ttl command
	client.ds.ttlCommands <- expiration{"WRONG COMMAND", "KEY", 15}
}

func TestLSet(t *testing.T) {
	client := setupTestClient()

	client.Exec("LPUSH", []string{"list", "0"})
	client.Exec("LPUSH", []string{"list", "1"})
	client.Exec("LPUSH", []string{"list", "2"})
	client.Exec("SET", []string{"x", "5"})

	runner(t, "LSET", client)

	client.Exec("LSET", []string{"doesntexist", "0", "14"})
	if client.err != errNoItem {
		t.Errorf("Set key non-existent list should give %#v, got: %#v", errNoItem, client.err)
	}
}

func TestLPush(t *testing.T) {
	client := setupTestClient()

	client.Exec("set", []string{"x", "15"})

	runner(t, "LPUSH", client)

	client.Exec("get", []string{"list"})
}

func TestLGet(t *testing.T) {
	client := setupTestClient()

	client.Exec("lpush", []string{"list", "value"})
	client.Exec("set", []string{"x", "0"})

	runner(t, "LGET", client)
}

func TestHSet(t *testing.T) {
	client := setupTestClient()

	client.Exec("set", []string{"x", "16"})

	runner(t, "HSET", client)
}

func TestHGet(t *testing.T) {
	client := setupTestClient()

	client.Exec("HSET", []string{"hash", "key", "value"})
	client.Exec("SET", []string{"x", "15"})

	runner(t, "HGET", client)
}
