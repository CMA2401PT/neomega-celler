module celler

go 1.25.5

require (
	github.com/df-mc/goleveldb v1.1.9
	neomega-blocks v0.0.0
	neomega-nbt v0.0.0
)

require github.com/pierrec/lz4/v4 v4.1.22

replace neomega-blocks => ../neomega-blocks

replace neomega-nbt => ../neomega-nbt

require (
	github.com/golang/snappy v0.0.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/klauspost/compress v1.13.6 // indirect
)
