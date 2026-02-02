FROM golang:1.25-trixie as builder

WORKDIR /app

COPY ["./go.mod", "./go.sum", "./"]

RUN go mod download

COPY ./ ./

RUN go build -o gcsproxy ./cmd/gcsproxy/main.go

FROM gcr.io/distroless/base-debian13:nonroot

COPY --from=builder /app/gcsproxy /
CMD ["/gcsproxy"]