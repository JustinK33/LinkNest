FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/linknest ./cmd/linknest

FROM alpine:3.23

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/linknest /app/linknest
USER appuser
EXPOSE 8080
CMD ["/app/linknest"]
