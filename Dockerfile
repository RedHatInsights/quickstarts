FROM registry.access.redhat.com/ubi9/go-toolset:9.6-1756993846 AS builder
WORKDIR $GOPATH/src/mypackage/myapp/
COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
COPY main.go main.go
COPY spec spec
COPY pkg pkg
COPY cmd cmd
COPY config config
COPY docs docs
ENV GO111MODULE=on
USER root
RUN go get -d -v
RUN make validate
RUN make test
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go/bin/quickstarts

# Build the migration binary.
RUN CGO_ENABLED=0 go build -o /go/bin/quickstarts-migrate cmd/migrate/migrate.go

 
FROM registry.access.redhat.com/ubi9-minimal:latest

COPY --from=builder /go/bin/quickstarts /usr/bin
COPY --from=builder /go/bin/quickstarts-migrate /usr/bin
COPY --from=builder /src/mypackage/myapp/spec/openapi.json /var/tmp
COPY --from=builder /src/mypackage/myapp/docs /docs

USER 1001


CMD ["quickstarts"]
EXPOSE 8000
