ARG TOOLS_DIR="/go/tools"

FROM golang:1.24.3-alpine@sha256:b4f875e650466fa0fe62c6fd3f02517a392123eea85f1d7e69d85f780e4db1c1 AS go-builder
ARG TOOLS_DIR

RUN mkdir ${TOOLS_DIR}

ADD prober prober/
ADD pager-duty pager-duty/

# the specific versions of these tools are in prober/hack/go.mod so that Dependabot can bump them for updates
RUN cd prober/hack && GOBIN=${TOOLS_DIR} go install -trimpath -ldflags="-w -s" tool

RUN cd prober && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/rate-limiting ./rate-limiting.go
RUN cd prober/ctlog && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/ctlog-sth ./ctlog-sth.go
RUN cd pager-duty && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/pager .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:188ddfb9e497f861177352057cb21913d840ecae6c843d39e00d44fa64daa51c

ARG TOOLS_DIR
COPY --from=go-builder ${TOOLS_DIR}/* /usr/local/bin/
