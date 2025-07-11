# rpz

Silly simple RPC benchmarking.

## Run (http)

Start the server

```bash
go run ./cmd/server -mode http
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

## Run (gRPC)

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

## TLS

The certicicates in ./cert were generated with this command:

```bash
openssl req -x509 -newkey rsa:2048 \
  -keyout certs/dev_key.pem -out certs/dev_cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost"
```
