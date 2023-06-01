module github.com/pieroforfora/atomicswapper/lib/btcatomicswap

require (
	github.com/btcsuite/btcd v0.0.0-20181013004428-67e573d211ac
	github.com/btcsuite/btcutil v0.0.0-20180706230648-ab6388e0c60a
	github.com/btcsuite/btcwallet v0.0.0-20181017015332-c4dd27e481f9
	golang.org/x/crypto v0.0.0-20220126234351-aa10faf2a1f8
)

require (
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd // indirect
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/sethvargo/go-envconfig v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/pieroforfora/atomicswapper v1.2.3

replace github.com/pieroforfora/atomicswapper => /home/pieroforfora/Devel/atomicswapper

go 1.18
