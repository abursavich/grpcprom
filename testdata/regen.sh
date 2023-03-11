DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --go_out="$DIR" --go_opt=paths=source_relative \
    --go-grpc_out="$DIR" --go-grpc_opt=paths=source_relative \
    --proto_path="$DIR" "$DIR"/*/*.proto
