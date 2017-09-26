package inmemory

import (
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
	"time"
)

// persistenced manages saving inmemory data to disk
// to be able to restart server and restore all data
func (dataStore *DataStore) persistenced() {

	// use directory to place backups
	backupsDir := ".backups"

	// create directory if doesn't exist
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		os.Mkdir(backupsDir, 0755)
	}

	for {
		time.Sleep(backupInterval)

		// Store all the data in the file
		err := dataStore.ToFile(backupsDir)
		// log the result of saving the data
		if err == nil {
			log.Println("Backup created")
		} else {
			log.Println("Error creating backup", err)
		}

		// number of backups is defined by variable backupNumber
		backups, err := filepath.Glob(backupsDir + "/cache_data*.gob")
		if err != nil {
			log.Println(err)
		}
		// delete obsolete backups
		if len(backups) > backupNumber {
			// delete all backups except for the defined number
			for _, old := range backups[0 : len(backups)-backupNumber] {
				err := os.Remove(old)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

// ToFile writes all data from the data store.
// gob encoding is used for the process
// Only the data is stored, the caching order is omitted
func (dataStore *DataStore) ToFile(path string) error {

	timestamp := time.Now().Format("20060102150405")

	path += "/cache_data" + timestamp + ".gob"
	// create file to write data to
	backup, err := os.Create(path)

	if err != nil {
		log.Println(err)
		return err
	}

	defer backup.Close()

	encCache := gob.NewEncoder(backup)

	dataStore.RLock()
	defer dataStore.RUnlock()

	return encCache.Encode(dataStore.values)
}

// FromFile reads gob file and restores data store.
// Also the cache data structure is also filled
func (dataStore *DataStore) FromFile(path string) error {

	backup, err := os.Open(path)

	if err != nil {
		log.Println(err)
		return err
	}

	defer backup.Close()

	decCache := gob.NewDecoder(backup)
	dataStore.Lock()
	decCache.Decode(&dataStore.values)

	// restore cache
	for _, item := range dataStore.values {
		el := dataStore.cache.PushFront(item)
		item.el = el
	}
	dataStore.Unlock()

	log.Printf("Restored %d values from backup %s\n", len(dataStore.values), path)

	return nil
}
