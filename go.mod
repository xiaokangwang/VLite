module github.com/xiaokangwang/VLite

go 1.13

require (
	//Apache 2.0
	github.com/FlowerWrong/netstack v0.0.0-20191009041010-964b57cc9e6e
	//BSD 3
	github.com/FlowerWrong/water v0.0.0-20180301012659-01a4eaa1f6f2
	//MIT
	github.com/boljen/go-bitmap v0.0.0-20151001105940-23cd2fb0ce7d
	//ISC License(BSD2 like)
	github.com/davecgh/go-spew v1.1.1
	//MIT
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	//Apache 2.0
	github.com/golang-collections/go-datastructures v0.0.0-20150211160725-59788d5eb259
	//BSD 3
	github.com/google/gopacket v1.1.17
	//Apache 2.0
	github.com/jacobsa/crypto v0.0.0-20190317225127-9f44e2d11115 // indirect
	//LGPL
	github.com/juju/ratelimit v1.0.1
	//MIT
	github.com/klauspost/cpuid v1.2.3 // indirect
	//MIT
	github.com/klauspost/reedsolomon v1.9.3
	//MIT
	github.com/lunixbochs/struc v0.0.0-20190916212049-a5c72983bc42
	//Apache 2.0
	github.com/mustafaturan/bus v1.0.2
	//Apache 2.0
	github.com/mustafaturan/monoton v1.0.0
	//MIT
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pion/dtls v1.5.4 // indirect
	github.com/pion/dtls/v2 v2.0.0-rc.7
	github.com/pion/logging v0.2.2
	github.com/pion/sctp v1.7.6
	github.com/secure-io/siv-go v0.0.0-20180922214919-5ff40651e2c4
	github.com/seiflotfy/cuckoofilter v0.0.0-20200416141329-862a88987de7
	github.com/stretchr/testify v1.5.1
	github.com/txthinking/runnergroup v0.0.0-20200327135940-540a793bb997 // indirect
	github.com/txthinking/socks5 v0.0.0-20200327133705-caf148ab5e9d
	github.com/txthinking/x v0.0.0-20200330144832-5ad2416896a9 // indirect
	github.com/xiaokangwang/water v0.0.0-20180524022535-3e9597577724
	github.com/xtaci/smux v1.5.12
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
)

replace github.com/FlowerWrong/netstack => ./vendor/github.com/FlowerWrong/netstack

replace github.com/FlowerWrong/water => ./vendor/github.com/FlowerWrong/water

replace github.com/golang-collections/go-datastructures => ./vendor/github.com/golang-collections/go-datastructures
