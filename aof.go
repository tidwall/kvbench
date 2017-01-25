package kvbench

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
)

var errInvalidLog = errors.New("invalid log")

type AOF struct {
	f     *os.File
	fsync bool
	buf   []byte
}

func openAOF(path string, fsync bool, cmd func(args [][]byte) error) (*AOF, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	err = func() error {
		rd := bufio.NewReader(f)
		var args [][]byte
		for {
			if c, err := rd.ReadByte(); err != nil {
				if err == io.EOF {
					break
				}
				return err
			} else if c != '*' {
				return errInvalidLog
			}
			line, err := rd.ReadString('\n')
			if err != nil {
				return err
			}
			if len(line) == 1 || line[len(line)-2] != '\r' {
				return errInvalidLog
			}
			n, err := strconv.ParseUint(line[:len(line)-2], 10, 64)
			if err != nil {
				return err
			}
			args = args[:0]
			for i := 0; i < int(n); i++ {
				if c, err := rd.ReadByte(); err != nil {
					return err
				} else if c != '$' {
					return errInvalidLog
				}
				line, err := rd.ReadString('\n')
				if err != nil {
					return err
				}
				if len(line) == 1 || line[len(line)-2] != '\r' {
					return errInvalidLog
				}
				n, err := strconv.ParseUint(line[:len(line)-2], 10, 64)
				if err != nil {
					return err
				}
				arg := make([]byte, int(n))
				if _, err := io.ReadFull(rd, arg); err != nil {
					return err
				}
				if c, err := rd.ReadByte(); err != nil {
					return err
				} else if c != '\r' {
					return errInvalidLog
				}
				if c, err := rd.ReadByte(); err != nil {
					return err
				} else if c != '\n' {
					return errInvalidLog
				}
				args = append(args, arg)
			}
			if len(args) == 0 {
				continue
			}
			if err := cmd(args); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		f.Close()
	}
	return &AOF{f: f, fsync: fsync}, nil
}
func (aof *AOF) Write(args ...[]byte) error {
	aof.BeginBuffer()
	aof.AppendBuffer(args...)
	return aof.WriteBuffer()
}

func (aof *AOF) BeginBuffer() {
	aof.buf = aof.buf[:0]
}

func (aof *AOF) AppendBuffer(args ...[]byte) {
	aof.buf = append(aof.buf, '*')
	aof.buf = strconv.AppendInt(aof.buf, int64(len(args)), 10)
	aof.buf = append(aof.buf, '\r', '\n')
	for _, arg := range args {
		aof.buf = append(aof.buf, '$')
		aof.buf = strconv.AppendInt(aof.buf, int64(len(arg)), 10)
		aof.buf = append(aof.buf, '\r', '\n')
		aof.buf = append(aof.buf, arg...)
		aof.buf = append(aof.buf, '\r', '\n')
	}
}

func (aof *AOF) WriteBuffer() error {
	_, err := aof.f.Write(aof.buf)
	if err != nil {
		return err
	}
	if aof.fsync {
		aof.f.Sync()
	}
	return err
}

func (aof *AOF) Close() error {
	aof.f.Close()
	return nil
}
