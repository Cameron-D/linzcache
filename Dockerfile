FROM golang:1 as builder
WORKDIR /go/src/github.com/Cameron_D/linzcache
RUN go get -d -v github.com/paulmach/orb
COPY main.go  .
RUN CGO_ENABLED=0 GOOS=linux go build -i -a -installsuffix cgo -o linzcache .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/Cameron_D/linzcache/linzcache .
ADD nz.geojson .
CMD ["./linzcache"]