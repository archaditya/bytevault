FROM golang:1.25.7-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o bytevault ./cmd/api

EXPOSE 8080

CMD ["./bytevault"]