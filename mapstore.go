package main

import (
	"strings"
	"sync"
	"time"

	"github.com/tidwall/match"
)

type mapStore struct {
	mu   sync.RWMutex
	keys map[string][]byte
	aof  *AOF
}

func newMapStore(path string, fsync bool) (*mapStore, error) {
	keys := make(map[string][]byte)
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
					keys[string(args[1])] = bcopy(args[2])
				}
			case "del":
				if len(args) >= 2 {
					delete(keys, string(args[1]))
				}
			case "flushdb":
				keys = make(map[string][]byte)
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
	return &mapStore{
		aof:  aof,
		keys: keys,
	}, nil
}

func (s *mapStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		s.aof.Close()
	}
	return nil
}

func (s *mapStore) PSet(keys, values [][]byte) error {
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
		s.keys[string(keys[i])] = bcopy(values[i])
	}
	return nil
}

func (s *mapStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var values [][]byte
	var oks []bool
	for i := range keys {
		v, ok := s.keys[string(keys[i])]
		if !ok {
			values = append(values, nil)
			oks = append(oks, false)
		} else {
			values = append(values, v)
			oks = append(oks, true)
		}
	}
	return values, oks, nil
}

func (s *mapStore) Set(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		if err := s.aof.Write([]byte("set"), key, value); err != nil {
			return err
		}
	}
	s.keys[string(key)] = bcopy(value)
	return nil
}

func (s *mapStore) Get(key []byte) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.keys[string(key)]
	return v, ok, nil
}

func (s *mapStore) Del(key []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.keys[string(key)]
	if ok {
		if s.aof != nil {
			if err := s.aof.Write([]byte("del"), key); err != nil {
				return false, err
			}
		}
		delete(s.keys, string(key))
	}
	return ok, nil
}

func (s *mapStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spattern := string(pattern)
	var keys [][]byte
	var vals [][]byte
	for key, value := range s.keys {
		if limit > -1 && len(keys) >= limit {
			break
		}
		if match.Match(key, spattern) {
			keys = append(keys, []byte(key))
			if withvalues {
				vals = append(vals, bcopy(value))
			}
		}
	}
	return keys, vals, nil
}

func (s *mapStore) FlushDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aof != nil {
		if err := s.aof.Write([]byte("flushdb")); err != nil {
			return err
		}
	}
	s.keys = make(map[string][]byte)
	return nil
}
