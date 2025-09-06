FROM golang:1.23.9-alpine AS test-runner

RUN apk update && apk add --no-cache bash

WORKDIR /app
COPY . .

RUN go mod tidy

# Run tests by default
CMD ["go", "test", "-v"]
