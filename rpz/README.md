# rpz

Silly simple RPC benchmarking.

## Run

Start the server

```bash
go run ./cmd/server
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
