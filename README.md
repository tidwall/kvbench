# KVBench

KVBench is a Redis server clone backed by a few different Go databases. 
It's intended to be used with the `redis-benchmark` command to test the performance of various Go databases.
It has support for redis pipelining.

Features:

- Databases
  - [BoltDB](https://github.com/boltdb/bolt)
  - [LevelDB](https://github.com/syndtr/goleveldb)
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


## Benchmark Results

*These benchmarks were run on a MacBook Pro 15" 2.8 GHz Intel Core i7 using Go 1.7.*

Store   | Pipeline | Clients | Persist | Fsync | SET/sec  | GET/sec
------- | -------- | ------- | ------- | ----- | -------  | -------
map     | 1        | 1       | yes     | yes   | 9246     | 35855
map     | 1        | 1       | yes     | no    | 30256    | 36670
map     | 1        | 1       | no      | no    | 37727    | 36592
btree   | 1        | 1       | yes     | yes   | 8610     | 32863
btree   | 1        | 1       | yes     | no    | 26992    | 33298
btree   | 1        | 1       | no      | no    | 34837    | 35968
bolt    | 1        | 1       | yes     | yes   | 4061     | 35698
bolt    | 1        | 1       | yes     | no    | 14295    | 34033
leveldb | 1        | 1       | yes     | yes   | 8774     | 37024
leveldb | 1        | 1       | yes     | no    | 23483    | 31654
redis   | 1        | 1       | yes     | yes   | 9361     | 34010
redis   | 1        | 1       | yes     | no    | 29439    | 30951
resis   | 1        | 1       | no      | no    | 33704    | 32879
map     | 1        | 256     | yes     | yes   | 12850    | 91962
map     | 1        | 256     | yes     | no    | 84495    | 99124
map     | 1        | 256     | no      | no    | 103928   | 98318
btree   | 1        | 256     | yes     | yes   | 12314    | 93510
btree   | 1        | 256     | yes     | no    | 73024    | 97819
btree   | 1        | 256     | no      | no    | 94188    | 101112
bolt    | 1        | 256     | yes     | yes   | 4386     | 88051
bolt    | 1        | 256     | yes     | no    | 21771    | 82135
leveldb | 1        | 256     | yes     | yes   | 83514    | 97389
leveldb | 1        | 256     | yes     | no    | 88051    | 101791
redis   | 1        | 256     | yes     | yes   | 118708   | 131302
redis   | 1        | 256     | yes     | no    | 132292   | 127893
resis   | 1        | 256     | no      | no    | 128435   | 121212
map     | 256      | 256     | yes     | yes   | 709723   | 2083333
map     | 256      | 256     | yes     | no    | 1076426  | 1960784
map     | 256      | 256     | no      | no    | 1324503  | 1733102
btree   | 256      | 256     | yes     | yes   | 350631   | 2309468
btree   | 256      | 256     | yes     | no    | 455580   | 2320185
btree   | 256      | 256     | no      | no    | 475285   | 2079002
bolt    | 256      | 256     | yes     | yes   | 115420   | 2487562
bolt    | 256      | 256     | yes     | no    | 193162   | 2732240
leveldb | 256      | 256     | yes     | yes   | 472813   | 2923976
leveldb | 256      | 256     | yes     | no    | 474833   | 2832861
redis   | 256      | 256     | yes     | yes   | 405022   | 979431
redis   | 256      | 256     | yes     | no    | 433651   | 942507
resis   | 256      | 256     | no      | no    | 690607   | 1004016





### 1M random keys, no pipelining, 1 client

benchmark command
```
redis-benchmark -p 6380 -q -t set,get -n 1000000 -r 1000000 -c 1 -P 1
```

#### map (in-memory)
persist + fsync
```
./kvbench --store=map
SET: 9246.76 requests per second
GET: 35855.14 requests per second
```
persist + no-fsync
```
./kvbench --store=map --fsync=false
SET: 30256.27 requests per second
GET: 36670.33 requests per second
```
nopersist
```
./kvbench --store=map --path=:memory:
SET: 37727.30 requests per second
GET: 36592.51 requests per second
```

#### btree (in-memory)
persist + fsync
```
./kvbench --store=btree
SET: 8610.74 requests per second
GET: 32863.39 requests per second
```
persist + no-fsync
```
./kvbench --store=btree --fsync=false
SET: 26992.01 requests per second
GET: 33298.93 requests per second
```
nopersist
```
./kvbench --store=btree --path=:memory:
SET: 34837.14 requests per second
GET: 35968.64 requests per second
```

#### bolt (on-disk)
persist + fsync
```
./kvbench --store=bolt
SET: 4061.14 requests per second
GET: 35698.99 requests per second
```
persist + no-fsync
```
./kvbench --store=bolt --fsync=false
SET: 14295.52 requests per second
GET: 34033.29 requests per second
```
nopersist
```
not available
```

#### leveldb (on-disk)
persist + fsync
```
./kvbench --store=leveldb
SET: 8774.78 requests per second
GET: 37024.70 requests per second
```
persist + no-fsync
```
./kvbench --store=leveldb --fsync=false
SET: 23483.55 requests per second
GET: 31654.59 requests per second
```
nopersist
```
not available
```

#### redis (in-memory)
persist + fsync
```
redis-server --port 6380 --appendonly yes --appendfsync always
SET: 9361.98 requests per second
GET: 34010.14 requests per second
```
persist + no-fsync
```
redis-server --port 6380 --appendonly yes --appendfsync no
SET: 29439.47 requests per second
GET: 30951.13 requests per second
```
nopersist
```
redis-server --port 6380 --appendonly no
SET: 33704.08 requests per second
GET: 32879.59 requests per second
```



### 1M random keys, no pipelining, 256 concurrent clients

benchmark command
```
redis-benchmark -p 6380 -q -t set,get -n 1000000 -r 1000000 -c 256 -P 1
```

#### map (in-memory)
persist + fsync
```
./kvbench --store=map
SET: 12850.68 requests per second
GET: 91962.48 requests per second
```
persist + no-fsync
```
./kvbench --store=map --fsync=false
SET: 84495.14 requests per second
GET: 99124.56 requests per second
```
nopersist
```
./kvbench --store=map --path=:memory:
SET: 103928.50 requests per second
GET: 98318.76 requests per second
```
#### btree (in-memory)
persist + fsync
```
./kvbench --store=btree
SET: 12314.21 requests per second
GET: 93510.38 requests per second
```
persist + no-fsync
```
./kvbench --store=btree --fsync=false
SET: 73024.68 requests per second
GET: 97819.67 requests per second
```
nopersist
```
./kvbench --store=btree --path=:memory:
SET: 94188.57 requests per second
GET: 101112.23 requests per second
```
#### bolt (on-disk)
persist + fsync
```
./kvbench --store=bolt
SET: 4386.21 requests per second
GET: 88051.42 requests per second
```
persist + no-fsync
```
./kvbench --store=bolt --fsync=false
SET: 21771.31 requests per second
GET: 82135.52 requests per second
```
nopersist
```
not available
```

#### leveldb (on-disk)
persist + fsync
```
./kvbench --store=leveldb
SET: 83514.28 requests per second
GET: 97389.95 requests per second
```
persist + no-fsync
```
./kvbench --store=leveldb --fsync=false
SET: 88051.42 requests per second
GET: 101791.52 requests per second
```
nopersist
```
not available
```

#### redis (in-memory)
persist + fsync
```
redis-server --port 6380 --appendonly yes --appendfsync always
SET: 118708.45 requests per second
GET: 131302.52 requests per second
```
persist + no-fsync
```
redis-server --port 6380 --appendonly yes --appendfsync no
SET: 132292.62 requests per second
GET: 127893.59 requests per second
```
nopersist
```
redis-server --port 6380 --appendonly no
SET: 128435.66 requests per second
GET: 121212.12 requests per second
```

### 1M random keys, 256 pipelining, 256 concurrent clients

benchmark command
```
redis-benchmark -p 6380 -q -t set,get -n 1000000 -r 1000000 -c 256 -P 256
```

#### map (in-memory)
persist + fsync
```
./kvbench --store=map
SET: 709723.19 requests per second
GET: 2083333.38 requests per second
```
persist + no-fsync
```
./kvbench --store=map --fsync=false
SET: 1076426.25 requests per second
GET: 1960784.38 requests per second
```
nopersist
```
./kvbench --store=map --path=:memory:
SET: 1324503.38 requests per second
GET: 1733102.12 requests per second
```
#### btree (in-memory)
persist + fsync
```
./kvbench --store=btree
SET: 350631.12 requests per second
GET: 2309468.75 requests per second
```
persist + no-fsync
```
./kvbench --store=btree --fsync=false
SET: 455580.88 requests per second
GET: 2320185.75 requests per second
```
nopersist
```
./kvbench --store=btree --path=:memory:
SET: 475285.16 requests per second
GET: 2079002.00 requests per second
```
#### bolt (on-disk)
persist + fsync
```
./kvbench --store=bolt
SET: 115420.13 requests per second
GET: 2487562.25 requests per second
```
persist + no-fsync
```
./kvbench --store=bolt --fsync=false
SET: 193162.06 requests per second
GET: 2732240.50 requests per second
```
nopersist
```
not available
```

#### leveldb (on-disk)
persist + fsync
```
./kvbench --store=leveldb
SET: 472813.25 requests per second
GET: 2923976.50 requests per second
```
persist + no-fsync
```
./kvbench --store=leveldb --fsync=false
SET: 474833.81 requests per second
GET: 2832861.25 requests per second
```
nopersist
```
not available
```

#### redis (in-memory)
persist + fsync
```
redis-server --port 6380 --appendonly yes --appendfsync always
SET: 405022.25 requests per second
GET: 979431.88 requests per second
```
persist + no-fsync
```
redis-server --port 6380 --appendonly yes --appendfsync no
SET: 433651.34 requests per second
GET: 942507.06 requests per second
```
nopersist
```
redis-server --port 6380 --appendonly no
SET: 690607.75 requests per second
GET: 1004016.06 requests per second
```

## Contact
Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

KVBench source code is available under the MIT [License](/LICENSE).


