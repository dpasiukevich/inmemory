package inmemory

import (
	"fmt"
	"strconv"
	"time"
)

// get fetches Item pointer from the data store.
func (dataStore *DataStore) get(key string) (*Item, bool) {
	value, ok := dataStore.values[key]
	if ok {
		return value, true
	}
	return nil, false
}

func (dataStore *DataStore) set(key string, value *Item) {
	dataStore.values[key] = value
}

// remove item from the data store by the given key.
// the item is also removed from cache
func (dataStore *DataStore) remove(key string) error {
	item, ok := dataStore.get(key)

	if !ok {
		return errNoItem
	}

	cacheEl := item.el

	// break the link from the Item for element in the cache for safe removal
	item.el = nil
	dataStore.cache.Remove(cacheEl)
	delete(dataStore.values, key)
	return nil
}

// Get command retrieves string value by given key from the data store.
// If item was successfully read, it will be updated as the most recently used in cache.
// Arguments are read from Args field of client object.
func Get(client *Client) {

	dataStore := client.Ds

	if len(client.Args) != 1 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.Err = errNoItem
		return
	}

	// convert item's value to the string
	result, ok := item.Value.(string)

	if !ok {
		client.Err = errNotString
		return

	}

	client.Reply = result

	// updating cache to set the current item as the most recently used
	dataStore.cache.MoveToFront(item.el)
}

// Set command will set string value by given key and value in the data store.
// If item was successfully set, it will be updated as the most recently used in cache.
// Arguments are read from Args field of client object.
func Set(client *Client) {

	dataStore := client.Ds

	if len(client.Args) < 2 || len(client.Args) > 3 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]
	value := client.Args[1]

	// expire is the time to live in seconds
	var expire int64

	// if ttl is set by user
	if len(client.Args) == 3 {

		// parse ttl and check it for correctness
		expire, err := strconv.ParseInt(client.Args[2], 10, 64)
		if err != nil {
			client.Err = errTTLFormat
			return
		}
		if expire < 0 {
			client.Err = errTTLValue
			return
		}
	} else {
		// use default expiration time
		expire = defaultExpiration
	}

	if expire != 0 {
		expire += time.Now().Unix()
	}

	item := &Item{
		Value:      value,
		Expiration: expire,
		el:         nil,
	}

	dataStore.Lock()
	defer dataStore.Unlock()

	dataStore.set(key, item)

	// update the cache
	el := dataStore.cache.PushFront(key)
	item.el = el

	client.Reply = "OK"
}

// Size command return number of all keys in the data store.
func Size(client *Client) {

	if len(client.Args) != 0 {
		client.Err = errArgumentNumber
		return
	}

	dataStore := client.Ds

	dataStore.RLock()
	defer dataStore.RUnlock()

	// convert int number of values to the string
	client.Reply = strconv.Itoa(len(dataStore.values))
}

// Remove element from the data store by given key.
func Remove(client *Client) {

	if len(client.Args) != 1 {
		client.Err = errArgumentNumber
		return
	}

	dataStore := client.Ds
	key := client.Args[0]

	dataStore.Lock()
	defer dataStore.Unlock()

	err := dataStore.remove(key)

	if err == nil {
		client.Reply = "OK"
	} else {
		client.Err = err
	}
}

// Keys which are currently in the data store.
// Result of the command is the string, containing all the keys.
func Keys(client *Client) {

	if len(client.Args) != 0 {
		client.Err = errArgumentNumber
		return
	}

	dataStore := client.Ds

	dataStore.RLock()
	defer dataStore.RUnlock()

	data := &dataStore.values
	res := make([]string, len(*data))

	// fill list of strings with current keys
	i := 0
	for k := range *data {
		res[i] = k
		i++
	}

	// convert list of strings to the one string
	client.Reply = fmt.Sprint(res)
}

// TTL updates ttl value of the item.
func TTL(client *Client) {

	if len(client.Args) != 2 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]

	// parse expiration time and check it for correctness
	expire, err := strconv.ParseInt(client.Args[1], 10, 64)
	if err != nil {
		client.Err = errTTLFormat
		return
	}

	if expire < 0 {
		client.Err = errTTLValue
		return
	}

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	if !ok {
		client.Err = errNoItem
		return
	}

	// set the expiration time
	item.Expiration = time.Now().Unix() + expire
	client.Reply = "OK"
}

// LSet updates item in the list object.
// This command is checking the index being in the range,
// so you cannot insert new value in the list, only update existing ones.
// Arguments are read from Args field of client object.
// List item will be updated as the most recently used in the cache.
func LSet(client *Client) {

	if len(client.Args) != 3 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]

	// get the index of the list and check it for correctness
	index, err := strconv.Atoi(client.Args[1])
	if err != nil {
		client.Err = errIndexFormat
		return
	}

	value := client.Args[2]

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.Err = errNoItem
		return
	}

	// convert the item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.Err = errNotList
		return
	}
	if index >= len(list) || index < 0 {
		client.Err = errIndexRange
		return
	}

	list[index] = value

	// update the cache
	dataStore.cache.MoveToFront(item.el)

	client.Reply = "OK"
}

// LPush is used to push value in the list.
// If there is no list, the command will create new one.
// Arguments are read from Args field of client object.
// List item will be updated as the most recently used in the cache.
func LPush(client *Client) {
	if len(client.Args) != 2 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]
	value := client.Args[1]

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	// create new list, if there is none
	if !ok {
		newItem := &Item{
			Value:      []string{value},
			Expiration: time.Now().Unix() + defaultExpiration,
			el:         nil,
		}
		dataStore.set(key, newItem)
		el := dataStore.cache.PushFront(key)
		newItem.el = el

		client.Reply = "OK"
		return
	}

	// convert existing item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.Err = errNotList
		return
	}
	item.Value = append(list, value)

	// update the cache
	dataStore.cache.MoveToFront(item.el)

	client.Reply = "OK"
}

// LGet returns value from the list item by given key.
// Arguments are read from Args field of client object.
// Successful run will put the value in the client's Reply field.
// Otherwise client's Reply field is empty string, and error in the client.Err field.
func LGet(client *Client) {

	if len(client.Args) != 2 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]
	index, err := strconv.Atoi(client.Args[1])
	if err != nil {
		client.Err = errIndexFormat
		return
	}

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.Err = errNoItem
		return
	}

	// convert existing item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.Err = errNotList
		return
	}
	if index >= len(list) || index < 0 {
		client.Err = errIndexRange
		return
	}

	// update the cache
	dataStore.cache.MoveToFront(item.el)
	client.Reply = list[index]
}

// HSet updates or creates the value in the hash item in the data store.
// If there is no hash item, it will be created.
// Arguments are read from Args field of client object.
// Hash item will be updated as the most recently used in the cache.
func HSet(client *Client) {

	if len(client.Args) != 3 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]
	hashKey := client.Args[1]
	value := client.Args[2]

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	// create new hash if it doesn't exist
	if !ok {
		newItem := &Item{
			Value: map[string]string{
				hashKey: value,
			},
			Expiration: time.Now().Unix() + defaultExpiration,
			el:         nil,
		}

		// set the value to new hash
		dataStore.set(key, newItem)

		// add new hash to the cache
		el := dataStore.cache.PushFront(key)
		newItem.el = el
		client.Reply = "OK"
		return
	}

	// convert existing item to the map type
	hash, ok := item.Value.(map[string]string)
	if !ok {
		client.Err = errNotHash
		return
	}

	hash[hashKey] = value
	dataStore.cache.MoveToFront(item.el)

	client.Reply = "OK"
}

// HGet retrieves value from hash by given key.
// If there is no such item, or hash doesn't exist - error will be thrown.
// Arguments are read from Args field of client object.
// Hash item will be updated as the most recently used in the cache.
func HGet(client *Client) {

	if len(client.Args) != 2 {
		client.Err = errArgumentNumber
		return
	}

	key := client.Args[0]
	hashKey := client.Args[1]

	dataStore := client.Ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	if !ok {
		client.Err = errNoItem
		return
	}

	// convert existing item to the hash type
	hash, ok := item.Value.(map[string]string)
	if !ok {
		client.Err = errNotHash
		return
	}
	result, ok := hash[hashKey]
	if !ok {
		client.Err = errNoKeyHash
		return
	}

	// update the cache
	dataStore.cache.MoveToFront(item.el)
	client.Reply = result
}