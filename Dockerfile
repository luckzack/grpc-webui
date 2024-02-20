FROM golang:1.13-alpine AS builder

ENV GO111MODULE=on

WORKDIR /src

COPY . ./
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o grpc-mocker main.go


FROM alpine

COPY ./index /index
COPY --from=builder /src/grpc-mocker ./
RUN mkdir /log
EXPOSE 6969
ENTRYPOINT ["./grpc-mocker"]
