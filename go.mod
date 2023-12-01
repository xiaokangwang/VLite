module github.com/xiaokangwang/VLite

go 1.13

require (
	github.com/FlowerWrong/water v0.0.0-20180301012659-01a4eaa1f6f2
	github.com/boljen/go-bitmap v0.0.0-20151001105940-23cd2fb0ce7d
	github.com/davecgh/go-spew v1.1.1
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/golang-collections/go-datastructures v0.0.0-20150211160725-59788d5eb259
	github.com/google/gopacket v1.1.17
	github.com/gorilla/websocket v1.4.2
	github.com/klauspost/cpuid v1.2.3 // indirect
	github.com/klauspost/reedsolomon v1.9.3
	github.com/lunixbochs/struc v0.0.0-20190916212049-a5c72983bc42
	github.com/mustafaturan/bus v1.0.2
	github.com/mustafaturan/monoton v1.0.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pion/dtls/v2 v2.0.0-rc.7
	github.com/pion/logging v0.2.2
	github.com/pion/sctp v1.7.6
	github.com/secure-io/siv-go v0.0.0-20180922214919-5ff40651e2c4
	github.com/seiflotfy/cuckoofilter v0.0.0-20200416141329-862a88987de7
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/txthinking/runnergroup v0.0.0-20200327135940-540a793bb997 // indirect
	github.com/txthinking/socks5 v0.0.0-20200327133705-caf148ab5e9d
	github.com/txthinking/x v0.0.0-20200330144832-5ad2416896a9 // indirect
	github.com/xtaci/smux v1.5.12
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/net v0.7.0
)

replace github.com/FlowerWrong/water => ./vendor2/github.com/FlowerWrong/water

replace github.com/golang-collections/go-datastructures => ./vendor2/github.com/golang-collections/go-datastructures
