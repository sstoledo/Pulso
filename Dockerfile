FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/worker ./cmd/worker \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S pulso \
    && adduser -S -G pulso pulso
COPY --from=build /out/api /out/worker /out/migrate /usr/local/bin/
USER pulso
EXPOSE 8080
CMD ["/usr/local/bin/api"]
