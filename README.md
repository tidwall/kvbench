# KVBench

KVBench is a Redis server clone backed by a few different Go databases. 
It's intended to be used with the `redis-benchmark` command to test the performance of various Go databases.
It has support for redis pipelining.

Features:

- Databases
  - [BoltDB](https://github.com/boltdb/bolt),
  - [LevelDB](https://github.com/syndtr/goleveldb),
  - map (in-memory) with [AOF persistence](https://redis.io/topics/persistence)
  - btree (in-memory) with [AOF persistence](https://redis.io/topics/persistence)
- Option to disable fsync
- Compatible with Redis clients


## Build

```
make
```

## Examples

Start server with various storage types:

```
./kvbench --store=map
./kvbench --store=btree
./kvbench --store=bolt
./kvbench --store=leveldb
```

Start server with non-default port:
```
./kvbench -p 6381 --store=btree
```


Start server with non-default path:
```
./kvbench --store=btree --path=mydata.db
```


Start server with fsync disabled:
```
./kvbench --store=btree --fsync=false
```

Start in-memory server with no disk persistence:
```
./kvbench --store=map --path=:memory:
```


## Benchmarking

Using the redis-benchmark tool, this is a simple command:

```
redis-benchmark -p 6380 -q -t set,get
```

This connects the server running at port 6380, and it benchmarks the `SET` and `GET` commands.
The results may look something like:

```
SET: 7902.23 requests per second
GET: 376923.25 requests per second
```

## Supported Redis Commands

```
SET key value
GET key
DEL key
KEYS pattern [LIMIT count]
FLUSHDB
QUIT
PING
SHUTDOWN
```

## Contact
Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

KVBench source code is available under the MIT [License](/LICENSE).


