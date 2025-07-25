FROM golang

WORKDIR /marketplace

COPY go.mod go.sum ./

RUN go mod download

COPY . .

EXPOSE 8080

CMD cd cmd && go build && ./cmd
