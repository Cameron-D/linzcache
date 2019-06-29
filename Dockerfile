FROM golang:1 as builder
WORKDIR /go/src/github.com/Cameron_D/mapcache
RUN go get -d -v github.com/paulmach/orb
COPY main.go  .
RUN CGO_ENABLED=0 GOOS=linux go build -i -a -installsuffix cgo -o mapcache .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/Cameron_D/mapcache/mapcache .
ADD nz.geojson .
CMD ["./mapcache"]