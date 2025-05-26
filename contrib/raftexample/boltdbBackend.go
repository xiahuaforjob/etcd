package main

import (
	bolt "go.etcd.io/bbolt"
	bolterrors "go.etcd.io/bbolt/errors"
)

type boltdbBackend struct {
	db *bolt.DB
}

func NewBoltdbBackend(path string) (*boltdbBackend, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Initialize the keys bucket if it doesn't exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("keys"))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &boltdbBackend{db: db}, nil
}

func (b *boltdbBackend) Put(key, value string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("keys"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), []byte(value))
	})
}

func (b *boltdbBackend) Get(key string) (string, error) {
	var value string
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return bolterrors.ErrBucketNotFound
		}
		val := bucket.Get([]byte(key))
		if val == nil {
			return bolterrors.ErrBucketNotFound
		}
		value = string(val)
		return nil
	})
	return value, err
}

func (b *boltdbBackend) Delete(key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return bolterrors.ErrBucketNotFound
		}
		return bucket.Delete([]byte(key))
	})
}
