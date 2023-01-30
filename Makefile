SRC := $(shell find . -name '*.go')

med: $(SRC)
	go build  -ldflags='-s -w' -o $@ cmd/$(@F)/main.go

internal/protobuf/med.pb.go: protobuf/med.proto
	protoc -I=protobuf --go_out=. protobuf/med.proto
