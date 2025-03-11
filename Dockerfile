FROM golang:latest

WORKDIR /app

# Install netcat for wait script
RUN apt-get update && apt-get install -y netcat-openbsd && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o swiftcodes ./cmd/swiftcodes

COPY schema.sql /app/
COPY config.toml /app/
COPY swift_codes.csv /app/

EXPOSE 8080

CMD ["/app/swiftcodes"]
