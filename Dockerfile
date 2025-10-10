ARG TOOLS_DIR="/go/tools"

FROM golang:1.25.2-alpine@sha256:182059d7dae0e1dfe222037d14b586ebece3ebf9a873a0fe1cc32e53dbea04e0 AS go-builder
ARG TOOLS_DIR

RUN mkdir ${TOOLS_DIR}

ADD prober prober/
ADD pager-duty pager-duty/

# the specific versions of these tools are in prober/hack/go.mod so that Dependabot can bump them for updates
RUN cd prober/hack && GOBIN=${TOOLS_DIR} go install -trimpath -ldflags="-w -s" tool

RUN cd prober && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/rate-limiting ./rate-limiting.go
RUN cd prober/ctlog && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/ctlog-sth ./ctlog-sth.go
RUN cd pager-duty && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/pager .
RUN cd prober/tiles-fsck && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/tiles-fsck .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0

ARG TOOLS_DIR
COPY --from=go-builder ${TOOLS_DIR}/* /usr/local/bin/
