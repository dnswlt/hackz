# rpz

Silly simple RPC and synchronization contention benchmarking.

## RPC

### Run (http)

Start the server

```bash
go run ./cmd/server -mode http -insecure
```

Then run a benchmark for GET:

```bash
curl -X POST http://localhost:8080/rpz/items \
  -H "Content-Type: application/json" \
  -d '{"id": "abc123", "name": "ManualInsert"}'

# Single item
wrk -t4 -c100 -d10s http://localhost:8080/rpz/items/abc123

# Multiple items
for i in {1..10000}; do
  curl -s -X POST http://localhost:8080/rpz/items \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"item$i\",\"name\":\"Preload\"}" > /dev/null
done

wrk -t4 -c100 -d10s -s wrk/get_items.lua http://localhost:8080
```

... or for POST:

```bash
wrk -t4 -c100 -d10s -s wrk/post_items.lua http://localhost:8080
```

Silly simple, right?

### Run (gRPC)

Start the server:

```bash
go run ./cmd/server -mode grpc
```

Run a load test (using TLS without client verification of the server's certificate chain and host name):

```bash
ghz \
  --proto proto/item_service.proto \
  --call rpz.ItemService.CreateItem \
  -d '{"id":"abc","name":"Name"}' \
  -c 100 -n 100000 \
  --skipTLS \
  localhost:9090
```

### TLS

The certicicates in ./cert were generated with this command:

```bash
openssl req -x509 -newkey rsa:2048 \
  -keyout certs/dev_key.pem -out certs/dev_cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost"
```

### Benchmark results

TL;DR: locally, >50k requests per second are no issue, both for http(s) and gRPC.
Over a local consumer 1 Gbps network, 50k+ RPS for gRPC and 30k RPS for http (no TLS)
are possible as well.

Running both client and server on localhost on a Macbook M1 Pro.

```bash
$ wrk -t4 -c100 -d10s -s wrk/post_items.lua http://localhost:8080
Running 10s test @ http://localhost:8080
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.31ms  244.29us   6.08ms   92.01%
    Req/Sec    18.81k     1.53k   40.81k    98.51%
  752225 requests in 10.10s, 148.71MB read
Requests/sec:  74479.71
Transfer/sec:     14.72MB

$ ghz \                                                          
  --proto proto/item_service.proto \
  --call rpz.ItemService.CreateItem \
  -d '{"id":"abc","name":"Name"}' \
  -c 100 -n 100000 \
  --skipTLS \
  localhost:9090

Summary:
  Count: 100000
  Total: 1.71 s
  Slowest: 11.76 ms
  Fastest: 0.12 ms
  Average: 1.21 ms
  Requests/sec: 58400.15

Response time histogram:
  0.116  [1]     |
  1.281  [59988] |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.445  [38608] |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  3.610  [1284]  |∎
  4.775  [18]    |
  5.940  [1]     |
  7.104  [0]     |
  8.269  [0]     |
  9.434  [48]    |
  10.599 [31]    |
  11.764 [21]    |

Latency distribution:
  10 % in 0.62 ms 
  25 % in 0.84 ms 
  50 % in 1.15 ms 
  75 % in 1.50 ms 
  90 % in 1.86 ms 
  95 % in 2.09 ms 
  99 % in 2.52 ms 

Status code distribution:
  [OK]   100000 responses 
```

## Contention

With ./cmd/counter/main.go you can benchmark a counter implementation
using `sync.Mutex`, `atomic.Int64` and a sharded atomic implementation
that distributes updates across N atomics and aggregates them at
query time.

```bash
Using 500 goroutines, 100000 iterations, 500 shards
Counter type *main.MutexCounter took 5.587 seconds. Counter value: 50000000 (ok=true)
Counter type *main.AtomicCounter took 3.690 seconds. Counter value: 50000000 (ok=true)
Counter type *main.ShardedAtomicCounter took 0.088 seconds. Counter value: 50000000 (ok=true)
```
