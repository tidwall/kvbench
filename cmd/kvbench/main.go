package main

import (
	"flag"
	"os"

	"github.com/tidwall/kvbench"
	"github.com/tidwall/redlog"
)

var log = redlog.New(os.Stderr)

func main() {
	var opts kvbench.Options
	flag.IntVar(&opts.Port, "p", 6380, "server port")
	flag.StringVar(&opts.Which, "store", "map", "store type: map,btree,bolt,leveldb")
	flag.BoolVar(&opts.Fsync, "fsync", true, "fsync")
	flag.StringVar(&opts.Path, "path", "", "database path or ':memory:' for none")
	flag.Parse()
	opts.Log = log
	if err := kvbench.Start(opts); err != nil {
		log.Warningf("%v", err)
		os.Exit(1)
	}
}
