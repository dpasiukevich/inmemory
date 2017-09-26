package inmemory

import (
	"strconv"
	"strings"
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

	dataStore := client.ds

	if len(client.args) != 1 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.err = errNoItem
		return
	}

	// convert item's value to the string
	result, ok := item.Value.(string)

	if !ok {
		client.err = errNotString
		return

	}

	client.reply = result

	// updating cache to set the current item as the most recently used
	dataStore.cache.MoveToFront(item.el)
}

// Set command will set string value by given key and value in the data store.
// If item was successfully set, it will be updated as the most recently used in cache.
// Arguments are read from Args field of client object.
func Set(client *Client) {

	dataStore := client.ds

	if len(client.args) < 2 || len(client.args) > 3 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]
	value := client.args[1]

	// expire is the time to live in seconds
	var expire int64

	// if ttl is set by user
	if len(client.args) == 3 {

		// parse ttl and check it for correctness
		expire, err := strconv.ParseInt(client.args[2], 10, 64)
		if err != nil {
			client.err = errTTLFormat
			return
		}
		if expire < 0 {
			client.err = errTTLValue
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
		Value: value,
		el:    nil,
	}

	dataStore.Lock()
	defer dataStore.Unlock()

	dataStore.set(key, item)
	dataStore.ttlCommands <- expiration{"SET", key, expire}

	// update the cache
	el := dataStore.cache.PushFront(key)
	item.el = el

	client.reply = "OK"
}

// Size command return number of all keys in the data store.
func Size(client *Client) {

	if len(client.args) != 0 {
		client.err = errArgumentNumber
		return
	}

	dataStore := client.ds

	dataStore.RLock()
	defer dataStore.RUnlock()

	// convert int number of values to the string
	client.reply = strconv.Itoa(len(dataStore.values))
}

// Remove element from the data store by given key.
func Remove(client *Client) {

	if len(client.args) != 1 {
		client.err = errArgumentNumber
		return
	}

	dataStore := client.ds
	key := client.args[0]

	dataStore.Lock()
	defer dataStore.Unlock()

	dataStore.ttlCommands <- expiration{"DELETE", key, 0}
	err := dataStore.remove(key)

	if err == nil {
		client.reply = "OK"
	} else {
		client.err = err
	}
}

// RemoveBatch of keys from the datastore.
func RemoveBatch(client *Client) {
	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	for _, key := range client.args {
		dataStore.ttlCommands <- expiration{"DELETE", key, 0}
		dataStore.remove(key)
	}

	client.reply = "OK"
}

// Keys which are currently in the data store.
// Result of the command is the string, containing all the keys.
func Keys(client *Client) {

	if len(client.args) != 0 {
		client.err = errArgumentNumber
		return
	}

	dataStore := client.ds

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
	client.reply = strings.Join(res, " ")
}

// TTL updates ttl value of the item.
func TTL(client *Client) {

	if len(client.args) != 2 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]

	// parse expiration time and check it for correctness
	expire, err := strconv.ParseInt(client.args[1], 10, 64)
	if err != nil {
		client.err = errTTLFormat
		return
	}

	if expire < 0 {
		client.err = errTTLValue
		return
	}

	dataStore := client.ds

	dataStore.ttlCommands <- expiration{"SET", key, time.Now().Unix() + expire}

	// set the expiration time
	//item.Expiration = time.Now().Unix() + expire
	client.reply = "OK"
}

// LSet updates item in the list object.
// This command is checking the index being in the range,
// so you cannot insert new value in the list, only update existing ones.
// Arguments are read from Args field of client object.
// List item will be updated as the most recently used in the cache.
func LSet(client *Client) {

	if len(client.args) != 3 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]

	// get the index of the list and check it for correctness
	index, err := strconv.Atoi(client.args[1])
	if err != nil {
		client.err = errIndexFormat
		return
	}

	value := client.args[2]

	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.err = errNoItem
		return
	}

	// convert the item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.err = errNotList
		return
	}
	if index >= len(list) || index < 0 {
		client.err = errIndexRange
		return
	}

	list[index] = value

	// update the cache
	dataStore.cache.MoveToFront(item.el)

	client.reply = "OK"
}

// LPush is used to push value in the list.
// If there is no list, the command will create new one.
// Arguments are read from Args field of client object.
// List item will be updated as the most recently used in the cache.
func LPush(client *Client) {
	if len(client.args) != 2 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]
	value := client.args[1]

	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	// create new list, if there is none
	if !ok {
		newItem := &Item{
			Value: []string{value},
			//Expiration: time.Now().Unix() + defaultExpiration,
			el: nil,
		}
		dataStore.set(key, newItem)
		el := dataStore.cache.PushFront(key)
		newItem.el = el

		client.reply = "OK"
		return
	}

	// convert existing item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.err = errNotList
		return
	}
	item.Value = append(list, value)

	// update the cache
	dataStore.cache.MoveToFront(item.el)

	client.reply = "OK"
}

// LGet returns value from the list item by given key.
// Arguments are read from Args field of client object.
// Successful run will put the value in the client's Reply field.
// Otherwise client's Reply field is empty string, and error in the client.Err field.
func LGet(client *Client) {

	if len(client.args) != 2 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]
	index, err := strconv.Atoi(client.args[1])
	if err != nil {
		client.err = errIndexFormat
		return
	}

	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)
	if !ok {
		client.err = errNoItem
		return
	}

	// convert existing item to the list type
	list, ok := item.Value.([]string)
	if !ok {
		client.err = errNotList
		return
	}
	if index >= len(list) || index < 0 {
		client.err = errIndexRange
		return
	}

	// update the cache
	dataStore.cache.MoveToFront(item.el)
	client.reply = list[index]
}

// HSet updates or creates the value in the hash item in the data store.
// If there is no hash item, it will be created.
// Arguments are read from Args field of client object.
// Hash item will be updated as the most recently used in the cache.
func HSet(client *Client) {

	if len(client.args) != 3 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]
	hashKey := client.args[1]
	value := client.args[2]

	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	// create new hash if it doesn't exist
	if !ok {
		newItem := &Item{
			Value: map[string]string{
				hashKey: value,
			},
			//Expiration: time.Now().Unix() + defaultExpiration,
			el: nil,
		}

		// set the value to new hash
		dataStore.set(key, newItem)

		// add new hash to the cache
		el := dataStore.cache.PushFront(key)
		newItem.el = el
		client.reply = "OK"
		return
	}

	// convert existing item to the map type
	hash, ok := item.Value.(map[string]string)
	if !ok {
		client.err = errNotHash
		return
	}

	hash[hashKey] = value
	dataStore.cache.MoveToFront(item.el)

	client.reply = "OK"
}

// HGet retrieves value from hash by given key.
// If there is no such item, or hash doesn't exist - error will be thrown.
// Arguments are read from Args field of client object.
// Hash item will be updated as the most recently used in the cache.
func HGet(client *Client) {

	if len(client.args) != 2 {
		client.err = errArgumentNumber
		return
	}

	key := client.args[0]
	hashKey := client.args[1]

	dataStore := client.ds

	dataStore.Lock()
	defer dataStore.Unlock()

	item, ok := dataStore.get(key)

	if !ok {
		client.err = errNoItem
		return
	}

	// convert existing item to the hash type
	hash, ok := item.Value.(map[string]string)
	if !ok {
		client.err = errNotHash
		return
	}
	result, ok := hash[hashKey]
	if !ok {
		client.err = errNoKeyHash
		return
	}

	// update the cache
	dataStore.cache.MoveToFront(item.el)
	client.reply = result
}
