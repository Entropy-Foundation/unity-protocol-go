# protoc -I proto/ proto/client.proto --go_out=plugins=grpc:proto

# protoc -I ./ api.proto --go_out=plugins=grpc:./

protoc -I api/ \
>     -I${GOPATH}/src \
>     --go_out=plugins=grpc:api \
>     api/api.proto

