package kvbench

import (
	"errors"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/tidwall/match"
)

var boltBucket = []byte("keys")

type boltStore struct {
	mu sync.RWMutex
	db *bolt.DB
}

func boltKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}
func newBoltStore(path string, fsync bool) (*boltStore, error) {
	if path == ":memory:" {
		return nil, errMemoryNotAllowed
	}
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	db.NoSync = !fsync
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(boltBucket)
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &boltStore{
		db: db,
	}, nil
}

func (s *boltStore) Close() error {
	s.db.Close()
	return nil
}

func (s *boltStore) PSet(keys, values [][]byte) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		for i := 0; i < len(keys); i++ {
			if err := b.Put(boltKey(keys[i]), values[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *boltStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var values [][]byte
	var oks []bool
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(boltBucket)
		for i := 0; i < len(keys); i++ {
			v := b.Get(boltKey(keys[i]))
			if v == nil {
				values = append(values, nil)
				oks = append(oks, false)
			} else {
				values = append(values, bcopy(v))
				oks = append(oks, true)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return values, oks, nil
}

func (s *boltStore) Set(key, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(boltBucket).Put(boltKey(key), value)
	})
}

func (s *boltStore) Get(key []byte) ([]byte, bool, error) {
	var v []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		v = tx.Bucket(boltBucket).Get(boltKey(key))
		return nil
	})
	return v, v != nil, err
}

func (s *boltStore) Del(key []byte) (bool, error) {
	var v []byte
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkey := boltKey(key)
		v = tx.Bucket(boltBucket).Get(bkey)
		return tx.Bucket(boltBucket).Delete(bkey)
	})
	return v != nil, err
}

func (s *boltStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	spattern := string(pattern)
	min, max := match.Allowable(spattern)
	bmin := []byte(min)
	var keys [][]byte
	var vals [][]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		if len(spattern) > 0 && spattern[0] == '*' {
			err := tx.Bucket(boltBucket).ForEach(func(key, value []byte) error {
				if limit > -1 && len(keys) >= limit {
					return errors.New("done")
				}
				skey := string(key[1:])
				if match.Match(skey, spattern) {
					keys = append(keys, []byte(skey))
					if withvalues {
						vals = append(vals, bcopy(value))
					}
				}
				return nil
			})
			if err != nil && err.Error() == "done" {
				err = nil
			}
			return err
		}
		c := tx.Bucket(boltBucket).Cursor()
		for key, value := c.Seek(bmin); key != nil; key, value = c.Next() {
			if limit > -1 && len(keys) >= limit {
				break
			}
			skey := string(key[1:])
			if skey >= max {
				break
			}
			if match.Match(skey, spattern) {
				keys = append(keys, []byte(skey))
				if withvalues {
					vals = append(vals, bcopy(value))
				}
			}
		}
		return nil
	})
	return keys, vals, err
}

func (s *boltStore) FlushDB() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(boltBucket); err != nil {
			return err
		}
		_, err := tx.CreateBucket(boltBucket)
		return err
	})
}
