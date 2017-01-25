package kvbench

import (
	"os"
	"sync"

	"github.com/cznic/kv"
)

type kvStore struct {
	mu    sync.RWMutex
	db    *kv.DB
	path  string
	fsync bool
}

func newKVStore(path string, fsync bool) (*kvStore, error) {
	if path == ":memory:" {
		return nil, errMemoryNotAllowed
	}
	db, err := kv.Create(path, &kv.Options{})
	if err != nil {
		return nil, err
	}

	return &kvStore{
		db:    db,
		path:  path,
		fsync: fsync,
	}, nil
}

func (s *kvStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Close()
	return nil
}
func (s *kvStore) PSet(keys, values [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.db.BeginTransaction(); err != nil {
		return err
	}
	defer s.db.Rollback()
	for i := range keys {
		if err := s.db.Set(keys[i], values[i]); err != nil {
			return err
		}
	}
	return s.db.Commit()
}

func (s *kvStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var values [][]byte
	var oks []bool
	for i := range keys {
		value, ok, err := s.Get(keys[i])
		if err != nil {
			return nil, nil, err
		}
		values = append(values, value)
		oks = append(oks, ok)
	}
	return values, oks, nil
}

func (s *kvStore) Set(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Set(key, value)
}

func (s *kvStore) Get(key []byte) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, err := s.db.Get(nil, key)
	if err != nil {
		return nil, false, err
	}
	return v, v != nil, nil
}

func (s *kvStore) Del(key []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.db.Get(nil, key)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	err = s.db.Delete(key)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *kvStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	return nil, nil, nil
	/*
		s.mu.RLock()
		defer s.mu.RUnlock()
		spattern := string(pattern)
		min, max := match.Allowable(spattern)
		bmin := []byte(min)
		var keys [][]byte
		var vals [][]byte
		useMax := !(len(spattern) > 0 && spattern[0] == '*')
		s.db.BeginTransaction()
		iter := s.db.NewIterator(nil, nil)
		for ok := iter.Seek(bmin); ok; ok = iter.Next() {
			if limit > -1 && len(keys) >= limit {
				break
			}
			key := iter.Key()
			value := iter.Value()
			skey := string(key)
			if useMax && skey >= max {
				break
			}
			if match.Match(skey, spattern) {
				keys = append(keys, []byte(skey))
				if withvalues {
					vals = append(vals, bcopy(value))
				}
			}
		}
		iter.Release()
		err := iter.Error()
		if err != nil {
			return nil, nil, err
		}
		return keys, vals, nil
	*/
}

func (s *kvStore) FlushDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Close()
	os.RemoveAll(s.path)
	s.db = nil
	db, err := kv.Create(s.path, &kv.Options{})
	if err != nil {
		return err
	}
	s.db = db
	return nil
}
