FROM golang:1.26 AS build
WORKDIR /app
COPY . .
RUN go mod download && CGO_ENABLED=0 go build -a -o iptracker -v ./...

FROM debian:trixie AS src_os
RUN apt update && apt install -y ca-certificates

FROM scratch
COPY containerfs/ .
COPY --from=build --chmod=555 /app/iptracker .
COPY --from=src_os /etc/ssl /etc/ssl
USER appuser
ENTRYPOINT [ "./iptracker" ]