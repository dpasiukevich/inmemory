Inmemory database
=======================

[![Go Doc](https://godoc.org/github.com/pasiukevich/inmemory?status.svg)](https://godoc.org/github.com/pasiukevich/inmemory)
[![Go Report Card](https://goreportcard.com/badge/github.com/pasiukevich/inmemory)](https://goreportcard.com/report/github.com/pasiukevich/inmemory)

About
-----

The in-memory database with caching, written in Golang. 

Install
-------
`go get github.com/pasiukevich/inmemory`

To run in the container, clone this repo and run:

`docker-compose up`

`go run client.go -addr=127.0.0.1:3050`

Features
--------

 - data types: string, list, hash
 - data clustering using consistent hashing
 - LRU caching
 - persistence to disk
 - tls protocol

Usage
-----

Running the server and client:
 - cd server/
 - go run server.go
 - cd client/
 - go run client.go

Package consist of 3 entities: 
  - proxy server
  - data server
  - client

Proxy server distributes the keys between data servers. So if client is working through the proxy server, all the requests will be distributes across the cluster.

Proxy server get the list of data servers from file (by default: cluster/servers.json). It dynamically disables routing to the data server, if it's removed from servers.json file. Proxy server uses connection pool to serve the requsts faster.

Also client can connect to the data server directly.

Data server options: 
```
  -addr string
    	Address to listen. (default "127.0.0.1:9443")
  -backup string
    	Path to file with backup in gob format. Used to restore previous state of server.
  -cert string
    	Server certificate filepath. (default "server.crt")
  -key string
    	Server key filepath. (default "server.key")
```

Proxy server options: 
```
  -addr string
    	Address to listen. (default "127.0.0.1:9443")
  -cert string
    	Server certificate filepath. (default "server.crt")
  -conns int
      Number of connections to keep for each data server (default 10)
  -key string
    	Server key filepath. (default "server.key")
  -servers string
      Path to file with the list of data servers (default "servers.json")
```

Client options:

```
  -addr string
    	Address to listen/connect. (default "127.0.0.1:9443")
```


Available commands:
- set key value
- get key
- lpush my_list value
- lset my_list 0 value
- lget my_list 0
- hset my_hash key value
- hget my_hash key
- size
- keys
- remove key
- ttl key 30 

Benchmarks
---------
```
BenchmarkGet-4     	20000000	        76.2 ns/op
BenchmarkSet-4     	 2000000	       565 ns/op
BenchmarkLPush-4   	 5000000	       305 ns/op
BenchmarkLGet-4    	20000000	        93.5 ns/op
BenchmarkHSet-4    	20000000	       105 ns/op
BenchmarkHGet-4    	20000000	        83.6 ns/op
```