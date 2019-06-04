FROM golang:1.12
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/wish/tcp-mux-proxy/
COPY . /go/src/github.com/wish/tcp-mux-proxy/
RUN dep ensure
RUN  GOOS=linux go build -a ./cmd/tcp-mux-proxy/


FROM alpine:3.9
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/wish/tcp-mux-proxy/tcp-mux-proxy .
CMD /root/tcp-mux-proxy
