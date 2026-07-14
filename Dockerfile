################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder AS builder

USER 0

WORKDIR /workspace

# Cache deps before copying source so that we do not need to re-download for every build
COPY go.mod go.sum ./

# Fetch dependencies
RUN go mod download

# Copy source files
COPY Makefile Makefile
COPY oapi-codegen.yaml oapi-codegen.yaml
COPY main.go main.go
COPY spec spec
COPY pkg pkg
COPY cmd cmd
COPY config config
COPY docs docs

# Generate API code and validate
RUN make generate
RUN make validate-api
RUN make openapi-json
RUN make validate

# Run tests
RUN make test

# Build all binaries
RUN CGO_ENABLED=1 go build -ldflags "-w -s" -buildvcs=false -o quickstarts
RUN CGO_ENABLED=1 go build -ldflags "-w -s" -o quickstarts-migrate cmd/migrate/migrate.go

############################
# STEP 2 build a small image
############################
FROM registry.access.redhat.com/hi/go:latest-fips

COPY --from=builder /workspace/quickstarts /usr/bin/
COPY --from=builder /workspace/quickstarts-migrate /usr/bin/
COPY --from=builder /workspace/spec/openapi.json /var/tmp
COPY --from=builder /workspace/docs /docs

ENV QUICKSTARTS_CONTENT_DIR=/docs

USER 1001

CMD ["quickstarts"]
EXPOSE 8000
