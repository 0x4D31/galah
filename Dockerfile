FROM golang:latest
COPY . .
RUN <<EOF
go mod download
go build
EOF
ENTRYPOINT ./galah -v