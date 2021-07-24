FROM golang@sha256:9cc582fad973e17fb3769737e373ac28c7a912991e3bb874d09facbfc0293260 as builder
WORKDIR /app

COPY . .
RUN go build

EXPOSE 5000

CMD ["go","run","main.go"]
