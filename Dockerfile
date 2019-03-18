FROM golang:1.11-rc-stretch

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go install -v ./...

CMD ["cloudinary-cleaner"]
