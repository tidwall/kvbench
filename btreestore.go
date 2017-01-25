package main

import (
	"strings"
	"sync"
	"time"

	"github.com/tidwall/btree"
	"github.com/tidwall/match"
)

type btreeStore struct {
	mu  sync.RWMutex
	tr  *btree.BTree
	aof *AOF
}

type btreeItem struct {
	key   string
	value []byte
}

func (a *btreeItem) Less(v btree.Item, ctx interface{}) bool {
	return a.key < v.(*btreeItem).key
}

func newBTreeStore(path string, fsync bool) (*btreeStore, error) {
	tr := btree.New(32, nil)
	var err error
	var aof *AOF
	if path == ":memory:" {
		log.Printf("persistance disabled")
	} else {
		var count int
		start := time.Now()
		aof, err = openAOF(path, fsync, func(args [][]byte) error {
			switch strings.ToLower(string(args[0])) {
			case "set":
				if len(args) >= 3 {
					tr.ReplaceOrInsert(
						&btreeItem{string(args[1]), bcopy(args[2])})
				}
			case "del":
				if len(args) >= 2 {
					tr.Delete(&btreeItem{string(args[1]), nil})
				}
			case "flushdb":
				tr = btree.New(32, nil)
			}
			count++
			return nil
		})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			log.Printf("loaded %d commands in %s", count, time.Since(start))
		}
	}
	return &btreeStore{
		aof: aof,
		tr:  tr,
	}, nil
}

func (s *btreeStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		s.aof.Close()
	}
	return nil
}

func (s *btreeStore) PSet(keys, values [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		s.aof.BeginBuffer()
		for i := range keys {
			s.aof.AppendBuffer([]byte("set"), keys[i], values[i])
		}
		err := s.aof.WriteBuffer()
		if err != nil {
			return err
		}
	}
	for i := range keys {
		s.tr.ReplaceOrInsert(&btreeItem{string(keys[i]), bcopy(values[i])})
	}
	return nil
}

func (s *btreeStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var values [][]byte
	var oks []bool
	for i := range keys {
		v := s.tr.Get(&btreeItem{string(keys[i]), nil})
		if v == nil {
			values = append(values, nil)
			oks = append(oks, false)
		} else {
			values = append(values, v.(*btreeItem).value)
			oks = append(oks, true)
		}
	}
	return values, oks, nil
}

func (s *btreeStore) Set(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		if err := s.aof.Write([]byte("set"), key, value); err != nil {
			return err
		}
	}
	s.tr.ReplaceOrInsert(&btreeItem{string(key), bcopy(value)})
	return nil
}

func (s *btreeStore) Get(key []byte) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.tr.Get(&btreeItem{string(key), nil})
	if v == nil {
		return nil, false, nil
	}
	return v.(*btreeItem).value, true, nil
}

func (s *btreeStore) Del(key []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.tr.Delete(&btreeItem{string(key), nil})
	if v != nil {
		if s.aof != nil {
			if err := s.aof.Write([]byte("del"), key); err != nil {
				return false, err
			}
		}
	}
	return v != nil, nil
}

func (s *btreeStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spattern := string(pattern)
	min, max := match.Allowable(spattern)
	var useMax bool
	pivot := &btreeItem{}
	if !(len(spattern) > 0 && spattern[0] == '*') {
		pivot.key = min
		useMax = true
	}
	var keys [][]byte
	var vals [][]byte
	s.tr.AscendGreaterOrEqual(pivot, func(v btree.Item) bool {
		if limit > -1 && len(keys) >= limit {
			return false
		}
		a := v.(*btreeItem)
		if useMax && a.key >= max {
			return false
		}
		if match.Match(a.key, spattern) {
			keys = append(keys, []byte(a.key))
			if withvalues {
				vals = append(keys, []byte(a.value))
			}
		}
		return true
	})
	return keys, vals, nil
}

func (s *btreeStore) FlushDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		if err := s.aof.Write([]byte("flushdb")); err != nil {
			return err
		}
	}
	s.tr = btree.New(32, nil)
	return nil
}
