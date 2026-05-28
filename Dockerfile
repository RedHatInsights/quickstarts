FROM registry.access.redhat.com/ubi9/go-toolset:1.26.2-1779959429@sha256:a2ba4645e7c424b08aa83ed7792e279683b0d33acbc5131b18183fd21e336c55 AS builder
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

 
FROM registry.access.redhat.com/ubi9-minimal:latest@sha256:5b74fce9d6e629942a0c6dc0f546c193e70d7f974d999a48c948c53dd3d36362

COPY --from=builder /go/bin/quickstarts /usr/bin
COPY --from=builder /go/bin/quickstarts-migrate /usr/bin
COPY --from=builder /src/mypackage/myapp/spec/openapi.json /var/tmp
COPY --from=builder /src/mypackage/myapp/docs /docs

USER 1001


CMD ["quickstarts"]
EXPOSE 8000
