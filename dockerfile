FROM golang:1.23.5-alpine3.21 AS builder

COPY . /youtubeToMp3
WORKDIR /youtubeToMp3

RUN go mod download && go get -u ./... && go mod tidy
RUN go build -o ./bin/bot cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=0 /youtubeToMp3/bin/bot .
COPY --from=0 /youtubeToMp3/configs configs/

CMD ["./bot"]