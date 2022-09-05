protoc-go:
	protoc -I=proto/ --go_out=. proto/*.proto

protoc-go-grpc:
	protoc -I=proto/ --go-grpc_out=. proto/*.proto

test:
	go test -cover -race ./...

server:
	go run cmd/server/main.go -port 50051

client:
	go run cmd/client/main.go -address 0.0.0.0:50051  

server1:
	go run cmd/server/main.go -port 50052 

server2:
	go run cmd/server/main.go -port 50053 

server1-tls:
	go run cmd/server/main.go -port 50052 -tls 

server2-tls:
	go run cmd/server/main.go -port 50053 -tls 


client1-tls:
	go run cmd/client/main.go -address 0.0.0.0:50052 -tls 

evans_cli:
	cd $$GOHOME/bin; evans -r repl -p 50051

cert:
	cd cert; ./gen.sh; cd ..



.PHONY:	protoc-go proto-go-grpc test server client evans_cli cert server1 server2 server1-tls server2-tls client1-tls


