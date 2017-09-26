// Package inmemory provides in-memory database implemetation with LRU caching.
// Supported types are string, list, hash.
package inmemory

import (
	"container/list"
	"errors"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// command table
	commands = map[string](func(*Client)){
		"SET":          Set,
		"GET":          Get,
		"SIZE":         Size,
		"REMOVE":       Remove,
		"REMOVE_BATCH": RemoveBatch,
		"KEYS":         Keys,
		"TTL":          TTL,
		"LSET":         LSet,
		"LPUSH":        LPush,
		"LGET":         LGet,
		"HSET":         HSet,
		"HGET":         HGet,
	}

	// default server configuration

	// number of backup files to keep
	backupNumber = 2
	// interval for backup service running
	backupInterval = 300 * time.Second
	// interval for memory cleanup service
	cleanupInterval = 5 * time.Second
	// default expiration for the item in seconds
	defaultExpiration int64 = 1800
	// max heap memory for the application
	maxMemory = 5000000
	// memory check interval in seconds
	memoryCheckInterval = 5

	// Error objects used by application
	errNoSuchCommand  = errors.New("no such command")
	errArgumentNumber = errors.New("wrong number of arguments")
	errNoItem         = errors.New("no such item")
	errTTLFormat      = errors.New("ttl should be a number")
	errTTLValue       = errors.New("ttl should be >= 0")
	errIndexFormat    = errors.New("index should be a number")
	errIndexRange     = errors.New("index out of range")
	errNotString      = errors.New("not a string")
	errNotList        = errors.New("not a list")
	errNotHash        = errors.New("not a hash")
	errNoKeyHash      = errors.New("no such key in the hash")
)

// Item struct holds the actual user's item(string, list, hash).
// It has expiration in seconds, Unix time. Usually set via time.Now().Unix()
// el is the link to the position in cache, for the O(1) cache manipulations.
type Item struct {
	Value interface{}
	el    *list.Element
}

type Expiration struct {
	Command string
	Key     string
	Time    int64
}

// DataStore struct holds all values for this database with LRU caching
// RWMutex is required for the thread-safe data reading and modification.
type DataStore struct {
	sync.RWMutex
	values      map[string]*Item
	cache       *list.List
	ttlCommands chan Expiration
}

// Client struct holds all info about the client, the last executed command,
// its output, err and arguments.
// Each client has the connection to the specific data store. In future there
// can be possibility to switch data stores by the client.
type Client struct {
	ds    *DataStore
	cmd   string
	args  []string
	err   error
	reply string
}

// New creates new data store and starts workers for it. Current workers: ttld,
// persistenced and memoryd.
func New() *DataStore {

	dataStore := DataStore{
		values:      make(map[string]*Item),
		cache:       list.New(),
		ttlCommands: make(chan Expiration, 15),
	}

	go dataStore.ttld()
	go dataStore.persistenced()
	go dataStore.memoryd()

	return &dataStore
}

// NewClient creates client for the given datastore.
func NewClient(dataStore *DataStore) *Client {
	return &Client{
		ds:    dataStore,
		cmd:   "",
		reply: "",
	}
}

// Exec is the command wrapper, giving the client possibility to invoke any command
// by string name and any correct set of arguments. The result of invokation is stored
// in the client struct. On correct usage the client's state is updated.
func (client *Client) Exec(command string, args []string) (reply string, err error) {

	command = strings.ToUpper(command)

	cmd, ok := commands[command]

	if ok {
		client.reply = ""
		client.err = nil

		client.cmd = command
		client.args = args

		cmd(client)
		return client.reply, client.err
	}

	return "", errNoSuchCommand
}

// memoryd is the worker process cleaning the memory its exceeding the limit
// current implementation is a bit silly :).
// on each interval 20 least recently items are deleted.
func (dataStore *DataStore) memoryd() {
	var memStats runtime.MemStats

	threshold := uint64(float64(maxMemory) * 0.9)
	checkInterval := time.Duration(memoryCheckInterval)

	client := NewClient(dataStore)

	for {
		runtime.ReadMemStats(&memStats)

		// naive solution
		// remove 20 least recently used elements from memory
		if memStats.Alloc > threshold {
			unusedEntries := make([]string, 0)
			dataStore.Lock()
			for i := 0; i < 20; i++ {
				elPtr := dataStore.cache.Back()
				if elPtr != nil {
					key := elPtr.Value.(string)
					unusedEntries = append(unusedEntries, key)
				}
			}
			dataStore.Unlock()

			// remove unused entries
			if len(unusedEntries) > 0 {
				client.Exec("REMOVE_BATCH", unusedEntries)
			}
		}

		time.Sleep(checkInterval * time.Second)
	}
}

// ttld is a worker clearing items with exceeded ttl.
func (dataStore *DataStore) ttld() {

	ticker := time.NewTicker(cleanupInterval)
	expirations := make(map[string]int64)
	client := NewClient(dataStore)

	for {
		select {
		// catch all ttl related commands to keep data consistent
		case expiration := <-dataStore.ttlCommands:
			switch expiration.Command {
			case "DELETE":
				delete(expirations, expiration.Key)
			case "SET":
				expirations[expiration.Key] = expiration.Time
			default:
				log.Println("ttld: cannot process command", expiration.Command)
			}
		// initialize cleanup for entries
		case <-ticker.C:
			var expired []string
			currentTime := time.Now().Unix()
			for k, v := range expirations {
				if v < currentTime {
					expired = append(expired, k)
				}
			}

			// delete expired entries from the data store
			if len(expired) > 0 {
				client.Exec("REMOVE_BATCH", expired)
			}
		}
	}
}
