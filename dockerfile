FROM golang:1.22-alpine AS build_base
WORKDIR /go/iv3

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o iv3 *.go

FROM alpine:latest as service
WORKDIR /go/iv3

RUN apk add --no-cache tar wget

# Install InfluxDB CLI.
RUN wget https://dl.influxdata.com/influxdb/releases/influxdb2-client-2.7.1-linux-amd64.tar.gz && \
	tar xvzf ./influxdb2-client-2.7.1-linux-amd64.tar.gz && \
	cp influx /usr/local/bin/

COPY --from=build_base /go/iv3/iv3 .

CMD ./iv3 -iv3Env ${IV3_ENV} -influxdbToken ${INFLUXDB_TOKEN} -influxdbUrl ${INFLUXDB_URL}
