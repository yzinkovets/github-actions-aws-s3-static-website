FROM golang:1.23 AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/aws-s3-static-website .


FROM alpine:latest
COPY --from=builder /bin/aws-s3-static-website /bin/aws-s3-static-website
RUN apk --no-cache add ca-certificates
ENTRYPOINT ["/bin/aws-s3-static-website"]