FROM golang:latest
WORKDIR /galah
COPY . .
RUN <<EOF
go mod download
go build
EOF
ENTRYPOINT ["./galah"]