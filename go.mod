module celler

go 1.23.3

require (
	github.com/df-mc/goleveldb v1.1.9
	github.com/gorilla/websocket v1.5.3
	neomega_blocks v0.0.0
	neomega_nbt v0.0.0
)

replace neomega_blocks v0.0.0 => ../neomega-blocks

replace neomega_nbt v0.0.0 => ../neomega-nbt

require (
	github.com/andybalholm/brotli v1.1.1
	github.com/golang/snappy v0.0.1 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect

)
