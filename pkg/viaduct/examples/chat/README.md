# Generate Certificates
ca certificate and a cert/key pair is required to have a setup for examples/chat. Same cert/key pair can be used in both server and client.

	# Generete Root Key
	openssl genrsa -des3 -out ca.key 4096
	# Generate Root Certificate
	openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt
	# Generate Key
	openssl genrsa -out chat.key 2048
	# Generate csr, Fill required details after running the command
	openssl req -new -key chat.key -out chat.csr
	# Generate Certificate
	openssl x509 -req -in chat.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out char.crt -days 500 -sha256

# How to Run

    cd examples/chat
    go build

## quic
- start server

	./chat --cmd-type=server --key=./chat.key --cert=./chat.crt --ca=./ca.crt --type=quic --addr=localhost:9890
- start client

	./chat --cmd-type=client --key=./chat.key --cert=./chat.crt --ca=./ca.crt --type=quic --addr=localhost:9890

## websocket
- start server

	./chat --cmd-type=server --key=./chat.key --cert=./chat.crt --ca=./ca.crt --type=websocket --addr=localhost:9890
- start client

	./chat --cmd-type=client --key=./chat.key --cert=./chat.crt --ca=./ca.crt --type=websocket --addr=wss://localhost:9890/test
