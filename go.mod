module celler

go 1.25.5

require (
	github.com/df-mc/goleveldb v1.1.9
	github.com/pierrec/lz4/v4 v4.1.22
	neomega-blocks v0.0.0
	neomega-nbt v0.0.0
	neomega-packfs v0.0.0
)

require (
	github.com/golang/snappy v0.0.1 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
)

replace neomega-blocks => ../neomega-blocks

replace neomega-nbt => ../neomega-nbt

replace neomega-packfs => ../neomega-packfs
