module github.com/kixa/hll-go

go 1.17

require github.com/klauspost/cpuid/v2 v2.0.9 // indirect

require (
	github.com/zeebo/xxh3 v1.0.1
	google.golang.org/protobuf v1.27.1
)

retract (
    v1.0.0 // contains cc/go inconsistency
)