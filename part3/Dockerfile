FROM golang as builder
LABEL maintainer="Marcel Juhnke <marrat@marrat.de>"

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates bash

COPY --from=builder /app/port-scan-exporter /usr/local/bin

CMD [ "port-scan-exporter" ]