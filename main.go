package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/redcon"
	"github.com/tidwall/redlog"
)

var log = redlog.New(os.Stderr)

type Store interface {
	Close() error
	Set(key, value []byte) error
	PSet(keys, values [][]byte) error
	Get(key []byte) ([]byte, bool, error)
	PGet(keys [][]byte) ([][]byte, []bool, error)
	Del(key []byte) (bool, error)
	Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error)
	FlushDB() error
}

func main() {
	var port int
	var which string
	var fsync bool
	var path string
	flag.IntVar(&port, "p", 6380, "server port")
	flag.StringVar(&which, "store", "map", "store type: map,btree,bolt,leveldb")
	flag.BoolVar(&fsync, "fsync", true, "fsync")
	flag.StringVar(&path, "path", "", "database path or ':memory:' for none")
	flag.Parse()
	var store Store
	var err error
	switch which {
	default:
		err = fmt.Errorf("unknown store type: %v", which)
	case "map":
		if path == "" {
			path = "map.db"
		}
		store, err = newMapStore(path, fsync)
	case "btree":
		if path == "" {
			path = "btree.db"
		}
		store, err = newBTreeStore(path, fsync)
	case "bolt":
		if path == "" {
			path = "bolt.db"
		}
		store, err = newBoltStore(path, fsync)
	case "leveldb":
		if path == "" {
			path = "leveldb.db"
		}
		store, err = newLevelDBStore(path, fsync)
	}
	if err != nil {
		log.Warningf("%v", err.Error())
		os.Exit(1)
	}
	defer store.Close()
	go func() {
		log.Printf("started server on port %d", port)
		log.Printf("store type: %v, fsync: %v", which, fsync)
	}()
	err = redcon.ListenAndServe(fmt.Sprintf(":%d", port),
		func(conn redcon.Conn, cmd redcon.Command) {
			cmdp, keys, values, is := parsePipeline(conn, cmd)
			if !is {
				cmdp = cmdParse(cmd.Args[0])
			}
			switch cmdp {
			default:
				conn.WriteError(
					"ERR unknown command '" + string(cmd.Args[0]) + "'")
			case cmdPING:
				conn.WriteString("PONG")
			case cmdQUIT:
				conn.WriteString("OK")
				conn.Close()
			case cmdPSET:
				err := store.PSet(keys, values)
				for i := 0; i < len(keys); i++ {
					if err != nil {
						conn.WriteError(err.Error())
					} else {
						conn.WriteString("OK")
					}
				}
			case cmdPGET:
				values, oks, err := store.PGet(keys)
				if err != nil {
					for i := 0; i < len(keys); i++ {
						conn.WriteError(err.Error())
					}
				} else {
					for i := 0; i < len(keys); i++ {
						v, ok := values[i], oks[i]
						if !ok {
							conn.WriteNull()
						} else {
							conn.WriteBulk(v)
						}
					}
				}
			case cmdSET:
				if len(cmd.Args) != 3 {
					wrongArgs(conn, cmd.Args[0])
					return
				}
				err := store.Set(cmd.Args[1], cmd.Args[2])
				if err != nil {
					conn.WriteError(err.Error())
				} else {
					conn.WriteString("OK")
				}
			case cmdGET:
				if len(cmd.Args) != 2 {
					wrongArgs(conn, cmd.Args[0])
					return
				}
				v, ok, err := store.Get(cmd.Args[1])
				if err != nil {
					conn.WriteError(err.Error())
				} else if !ok {
					conn.WriteNull()
				} else {
					conn.WriteBulk(v)
				}
			case cmdDEL:
				if len(cmd.Args) != 2 {
					wrongArgs(conn, cmd.Args[0])
					return
				}
				ok, err := store.Del(cmd.Args[1])
				if err != nil {
					conn.WriteError(err.Error())
				} else if !ok {
					conn.WriteInt(0)
				} else {
					conn.WriteInt(1)
				}
			case cmdFLUSHDB:
				if len(cmd.Args) != 1 {
					wrongArgs(conn, cmd.Args[0])
					return
				}
				err := store.FlushDB()
				if err != nil {
					conn.WriteError(err.Error())
				} else {
					conn.WriteString("OK")
				}
			case cmdKEYS:
				if len(cmd.Args) < 2 {
					wrongArgs(conn, cmd.Args[0])
					return
				}
				var withvalues bool
				limit := -1
				for i := 2; i < len(cmd.Args); i++ {
					switch strings.ToLower(string(cmd.Args[i])) {
					case "withvalues":
						withvalues = true
					case "limit":
						i++
						if i == len(cmd.Args) {
							syntaxErr(conn)
							return
						}
						n, err := strconv.ParseInt(string(cmd.Args[i]), 10, 64)
						if err != nil || n < 0 {
							syntaxErr(conn)
							return
						}
						limit = int(n)
					}
				}
				keys, vals, err := store.Keys(cmd.Args[1], limit, withvalues)
				if err != nil {
					conn.WriteError(err.Error())
				} else {
					if withvalues {
						conn.WriteArray(len(keys) * 2)
					} else {
						conn.WriteArray(len(keys))
					}
					for i := 0; i < len(keys); i++ {
						conn.WriteBulk(keys[i])
						if withvalues {
							conn.WriteBulk(vals[i])
						}
					}
				}
			}
		}, nil, nil)
	if err != nil {
		log.Warningf("%v", err)
		os.Exit(1)
	}
}

type cmdType int

const (
	cmdUnknown cmdType = iota
	cmdFLUSHDB
	cmdKEYS
	cmdPING
	cmdQUIT
	cmdDEL
	cmdGET
	cmdSET

	cmdPSET
	cmdPGET
)

func cmdParse(cmd []byte) cmdType {
	switch len(cmd) {
	case 7:
		if (cmd[0] == 'F' || cmd[0] == 'f') &&
			(cmd[1] == 'L' || cmd[1] == 'l') &&
			(cmd[2] == 'U' || cmd[2] == 'u') &&
			(cmd[3] == 'S' || cmd[3] == 's') &&
			(cmd[4] == 'H' || cmd[4] == 'h') &&
			(cmd[5] == 'D' || cmd[5] == 'd') &&
			(cmd[6] == 'B' || cmd[6] == 'b') {
			return cmdFLUSHDB
		}
	case 4:
		if (cmd[0] == 'K' || cmd[0] == 'k') &&
			(cmd[1] == 'E' || cmd[1] == 'e') &&
			(cmd[2] == 'Y' || cmd[2] == 'y') &&
			(cmd[3] == 'S' || cmd[3] == 's') {
			return cmdKEYS
		}
		if (cmd[0] == 'P' || cmd[0] == 'p') &&
			(cmd[1] == 'I' || cmd[1] == 'i') &&
			(cmd[2] == 'N' || cmd[2] == 'n') &&
			(cmd[3] == 'G' || cmd[3] == 'g') {
			return cmdPING
		}
		if (cmd[0] == 'Q' || cmd[0] == 'q') &&
			(cmd[1] == 'U' || cmd[1] == 'u') &&
			(cmd[2] == 'I' || cmd[2] == 'i') &&
			(cmd[3] == 'T' || cmd[3] == 't') {
			return cmdQUIT
		}
	case 3:
		if (cmd[0] == 'D' || cmd[0] == 'd') &&
			(cmd[1] == 'E' || cmd[1] == 'e') &&
			(cmd[2] == 'L' || cmd[2] == 'l') {
			return cmdDEL
		}
		if (cmd[0] == 'G' || cmd[0] == 'g') &&
			(cmd[1] == 'E' || cmd[1] == 'e') &&
			(cmd[2] == 'L' || cmd[2] == 't') {
			return cmdGET
		}
		if (cmd[0] == 'S' || cmd[0] == 's') &&
			(cmd[1] == 'E' || cmd[1] == 'e') &&
			(cmd[2] == 'T' || cmd[2] == 't') {
			return cmdSET
		}
	}
	return cmdUnknown
}

func bcopy(b []byte) []byte {
	r := make([]byte, len(b))
	copy(r, b)
	return r
}
func wrongArgs(conn redcon.Conn, cmd []byte) {
	conn.WriteError(
		"ERR wrong number of arguments for '" + string(cmd) + "' command")
}
func syntaxErr(conn redcon.Conn) {
	conn.WriteError("ERR syntax error")
}
func parsePipeline(conn redcon.Conn, cmd redcon.Command) (
	cmdp cmdType, keys, values [][]byte, ok bool,
) {
	cmds := conn.PeekPipeline()
	if len(cmds) == 0 {
		return
	}
	// we have a pipeline
	cmdt := cmdParse(cmd.Args[0])
	switch cmdt {
	case cmdSET, cmdGET:
		cmdp = cmdt
		keys = append(keys, cmd.Args[1])
		if len(cmd.Args) == 3 {
			values = append(values, cmd.Args[2])
		}
		for _, cmd := range cmds {
			cmdt = cmdParse(cmd.Args[0])
			if len(cmd.Args) > 3 || cmdp != cmdt {
				return
			}
			keys = append(keys, cmd.Args[1])
			if len(cmd.Args) == 3 {
				values = append(values, cmd.Args[2])
			}
		}
		if cmdp != cmdUnknown {
			conn.ReadPipeline()
			if cmdp == cmdGET {
				cmdp = cmdPGET
			} else if cmdp == cmdSET {
				cmdp = cmdPSET
			}
			ok = true
			return
		} else {
			cmdp = cmdUnknown
			return
		}
	}
	return
}
