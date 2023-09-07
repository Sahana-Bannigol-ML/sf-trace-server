FROM golang:1.18
COPY metrics_collector/ /go/metrics_collector
WORKDIR /go/metrics_collector
RUN go mod tidy
RUN go build  -o /go/metrics metrics.go
RUN apt-get update -y
RUN apt-get install vim -y
RUN apt-get install curl -y
WORKDIR /go
RUN chmod +x metrics
ENTRYPOINT ["metrics"]