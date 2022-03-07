package stores

import (
	"fmt"

	c "di2e.net/cwbi/nsiv2-api/config"
	"github.com/boltdb/bolt"
	_ "github.com/jackc/pgx/stdlib"
)

type TempStore struct {
	store *bolt.DB
}

func InitTempStore(config c.AppConfig) (*TempStore, error) {
	store := TempStore{}
	err := store.Open(config)
	return &store, err
}

func (ts *TempStore) Open(appConfig c.AppConfig) error {
	db, err := bolt.Open(appConfig.TempStoragePath+"/temp.db", 0600, nil)
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("STATUS"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	ts.store = db
	return nil
}

func (ts *TempStore) Close() error {
	err := ts.store.Close()
	if err != nil {
		return err
	}
	return nil
}

func (ts *TempStore) PutStatus(guid string, status string) error {
	err := ts.store.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("STATUS")).Put([]byte(guid), []byte(status))
		if err != nil {
			return fmt.Errorf("could not update status for %s: %v", guid, err)
		}
		return nil
	})
	return err
}

func (ts *TempStore) GetStatus(guid string) (string, error) {
	var status string
	err := ts.store.View(func(tx *bolt.Tx) error {
		status = string(tx.Bucket([]byte("STATUS")).Get([]byte(guid)))
		return nil
	})
	if err != nil {
		return "", err
	} else {
		return status, nil
	}
}
