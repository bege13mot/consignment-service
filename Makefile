build: buildApi buildProxy buildSwagger

buildApi:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:$(GOPATH)/src/github.com/bege13mot/consignment-service \
		proto/consignment/consignment.proto

buildProxy:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:. \
		proto/consignment/consignment.proto

buildSwagger:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--swagger_out=logtostderr=true:. \
		proto/consignment/consignment.proto

# build:
# 	#protoc -I. --go_out=plugins=grpc
# 	protoc -I. --go_out=plugins=grpc:$(GOPATH)/src/github.com/testProject/shippy-consignment-service-new \
# 	  proto/consignment/consignment.proto
# 	# GOOS=linux GOARCH=amd64 go build
# 	# docker build -t shippy-consignment-service-new .
# 	# docker build -t ewanvalentine/consignment:latest .
# 	# docker push ewanvalentine/consignment:latest

# build:
# 	protoc -I/usr/local/include -I. \
#   	-I$(GOPATH)/src \
#   	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
# 		--go_out=plugins=grpc:$(GOPATH)/src/github.com/testProject/consignment-service \
# 		proto/consignment/consignment.proto
#
# buildProxy:
# 	protoc -I/usr/local/include -I. \
#   	-I$(GOPATH)/src \
#   	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
# 		--grpc-gateway_out=logtostderr=true:. \
# 		proto/consignment/consignment.proto

run:
	docker run -d --net="host" \
		-p 50052 \
		-e MICRO_SERVER_ADDRESS=:50052 \
		-e MICRO_REGISTRY=mdns \
		-e DISABLE_AUTH=true \
		consignment-service

deploy:
	sed "s/{{ UPDATED_AT }}/$(shell date)/g" ./deployments/deployment.tmpl > ./deployments/deployment.yml
	kubectl replace -f ./deployments/deployment.yml


# run:
# 	docker run -p 50051:50051 consignment-service
