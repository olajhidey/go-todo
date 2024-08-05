FROM golang:1.20-alpine AS builder


WORKDIR /app

COPY . .

RUN go mod download

# Install necessary C libraries for SQLite
RUN apk update && apk add --no-cache gcc musl-dev

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o app .

FROM alpine:latest

# Install necessary C libraries for SQLite
RUN apk update && apk add --no-cache sqlite-libs

WORKDIR /root/

COPY --from=builder /app/app .

# Copy the HTML folder into the container
COPY --from=builder /app/www ./www

EXPOSE 8080

CMD [ "./app" ]