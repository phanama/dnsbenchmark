FROM golang:1.16.5-alpine AS build

WORKDIR /go/src/github.com/test/dnsbenchmark
COPY . .

RUN go mod download
RUN go build -o /dnsbenchmark dnsbenchmark.go

FROM alpine:3.11

RUN apk update && apk add ca-certificates
RUN addgroup -S user && adduser -S user -G user
USER user
COPY --from=build /dnsbenchmark /dnsbenchmark
