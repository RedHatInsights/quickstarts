FROM registry.redhat.io/rhel8/go-toolset:1.20.12-5.1712568462 AS builder
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .
ENV GO111MODULE=on
USER root
RUN go get -d -v
RUN make validate
RUN make test
RUN CGO_ENABLED=0 go build -buildvcs=false -o /go/bin/quickstarts

# Build the migration binary.
RUN CGO_ENABLED=0 go build -o /go/bin/quickstarts-migrate cmd/migrate/migrate.go

 
FROM registry.redhat.io/ubi8-minimal:latest

COPY --from=builder /go/bin/quickstarts /usr/bin
COPY --from=builder /go/bin/quickstarts-migrate /usr/bin
COPY --from=builder /src/mypackage/myapp/spec/openapi.json /var/tmp
COPY --from=builder /src/mypackage/myapp/docs /docs

USER 1001


CMD ["quickstarts"]
EXPOSE 8000
