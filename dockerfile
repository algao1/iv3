FROM golang:1.20-alpine AS build_base
WORKDIR /go/iv3

COPY go.mod go.sum ./
RUN go mod download

FROM build_base as service_builder
WORKDIR /go/iv3

COPY . ./
RUN go build -o iv3 *.go

FROM golang:1.20-alpine as service
WORKDIR /go/iv3

COPY --from=service_builder /go/iv3/iv3 .
COPY --from=service_builder /go/iv3/config.yaml .
COPY --from=service_builder /go/iv3/server.crt .
COPY --from=service_builder /go/iv3/server.key .

CMD ./iv3 -influxdbToken ${INFLUXDB_TOKEN} -influxdbUrl ${INFLUXDB_URL}