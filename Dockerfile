FROM golang:1.8

WORKDIR /go/src/registry-sample
COPY . .

RUN go build .
RUN chmod +x registry-sample

EXPOSE 5000

CMD ["./registry-sample"]