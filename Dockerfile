#The first image to setup environment variable in container
FROM alpine:latest as os 
#Environment Variable to connected Database
ENV DB_CERT_PATH=
ENV COMPONENT=ClientAPI
ENV GRPC_PORT=
ENV HTTP_PORT=
ENV PGPASSWORD=db
ENV CONTAINER_REPO=
ENV CONTAINER_REPO_STAGING=
ENV CONTAINER_REPO_PROD=
ENV REPO_URL=
ENV REPO_PATH=
ENV CPU_REQ=100m
ENV MEMORY_REQ=100Mi
ENV CPU_LIMIT=200m
ENV MEMORY_LIMIT=200Mi
ENV TEST_DB_HOST=testdb
ENV TEST_DB_USER=db
ENV TEST_DB_PORT=5432
ENV TEST_DB_NAME=testdb
ENV TEST_DB_PASS=db
ENV INTEGRASI_INQUIRY_SWITCHING_URL=
ENV TEST_WITHOUT_GRPC=0
ENV HTTPS_PROXY ${http_proxy}
ENV HTTP_PROXY ${http_proxy}
ENV GO111MODULE on
RUN apk update
RUN apk add curl git gcc musl-dev bash

#The second image to build go binary & copy some of needed application
FROM golang:1.15.2-alpine3.12 as build
RUN apk add --update bash
#Set current working directory inside the container
WORKDIR /app
# Copy all sourcecode from the Workspace Directory to the Working Directory inside the container
COPY . .
#Build API binary Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o deployments/docker/build/grpc cmd/grpc/main.go
#Copy binary apps, script apps, & certificate
COPY . /app

#The last image to packaging our application    
FROM os
WORKDIR /app
RUN apk add --no-cache tzdata
ENV TZ=Asia/Jakarta
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

#Starting application in container
RUN chmod +x /app/start.sh /app/wait-for-it.sh
ENTRYPOINT [ "/app/start.sh" ]
