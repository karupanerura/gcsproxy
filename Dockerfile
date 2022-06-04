FROM golang:1.18 as builder

WORKDIR /app

COPY ["./go.mod", "./go.sum", "./"]

RUN go mod download

COPY ./ ./

RUN go build -o app ./cmd/gcsproxy/main.go

FROM gcr.io/distroless/base-debian10

COPY --from=builder /app/app /
CMD ["/app"]