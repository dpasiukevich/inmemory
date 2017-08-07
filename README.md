Inmemory database
=======================

[![Go Doc](https://godoc.org/github.com/pasiukevich/inmemory?status.svg)](https://godoc.org/github.com/pasiukevich/inmemory)
[![Go Report Card](https://goreportcard.com/report/github.com/pasiukevich/inmemory)](https://goreportcard.com/report/github.com/pasiukevich/inmemory)

About
-----

The in-memory database with caching, written in Golang. 

Install
-------
`go get github.com/pasiukevich/inmemory`

To run in the container, clone this repo and run:

`docker-compose up inmemory`

Features
--------

 - data types: string, list, hash
 - LRU caching
 - persistence to disk
 - tls protocol

Usage
-----

1. run server/server.go
2. run client/client.go to connect

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