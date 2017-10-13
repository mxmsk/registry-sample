FROM golang:1.8

WORKDIR /go/src/registry-sample
COPY . .

RUN go build .
RUN chmod +x registry-sample

CMD ["/go/src/registry-sample"]
