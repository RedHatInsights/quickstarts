FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1780373831@sha256:49f5929f6674d75377902ddcc2f46baf7a5cfcaada2497ee43f66e090943afd6 AS builder
WORKDIR $GOPATH/src/mypackage/myapp/
COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
COPY oapi-codegen.yaml oapi-codegen.yaml
COPY main.go main.go
COPY spec spec
COPY pkg pkg
COPY cmd cmd
COPY config config
COPY docs docs
ENV GO111MODULE=on
ENV GOTOOLCHAIN=go1.26.3
USER root
RUN make generate
RUN make validate-api
RUN go get -d -v
RUN make openapi-json
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
