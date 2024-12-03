# build
FROM golang:latest as build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# run
FROM alpine:latest as run

RUN apk add --no-cache poppler-utils

WORKDIR /app

COPY --from=build /build/main .
COPY --from=build /build/tmp ./tmp
COPY --from=build /build/test-1.pdf ./test-1.pdf
COPY --from=build /build/test-2.pdf ./test-2.pdf

# Create output directory with appropriate permissions
RUN mkdir -p /app/output && chmod 777 /app/output

CMD ["./main"] 