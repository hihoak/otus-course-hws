FROM golang:1.19.1 as build

WORKDIR /app
COPY . .
RUN go mod download

CMD go test --tags=integration_tests /app/...