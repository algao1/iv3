FROM golang:1.20-alpine AS build_base
WORKDIR /go/iv3

COPY go.mod go.sum ./
RUN go mod download

FROM build_base as service_builder
WORKDIR /go/iv3

COPY . ./
RUN go build -o iv3 *.go

FROM alpine:latest as service
WORKDIR /go/iv3

RUN apk add --no-cache tar

# Install InfluxDB CLI.
RUN wget https://dl.influxdata.com/influxdb/releases/influxdb2-client-2.7.1-linux-amd64.tar.gz
RUN tar xvzf ./influxdb2-client-2.7.1-linux-amd64.tar.gz
RUN cp influx /usr/local/bin/

COPY --from=service_builder /go/iv3/iv3 .
COPY --from=service_builder /go/iv3/config.yaml .
COPY --from=service_builder /go/iv3/certfile.crt .
COPY --from=service_builder /go/iv3/keyfile.key .

CMD ./iv3 -influxdbToken ${INFLUXDB_TOKEN} -influxdbUrl ${INFLUXDB_URL}