ARG TOOLS_DIR="/go/tools"

FROM golang:1.24.5-alpine@sha256:daae04ebad0c21149979cd8e9db38f565ecefd8547cf4a591240dc1972cf1399 AS go-builder
ARG TOOLS_DIR

RUN mkdir ${TOOLS_DIR}

ADD prober prober/
ADD pager-duty pager-duty/

# the specific versions of these tools are in prober/hack/go.mod so that Dependabot can bump them for updates
RUN cd prober/hack && GOBIN=${TOOLS_DIR} go install -trimpath -ldflags="-w -s" tool

RUN cd prober && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/rate-limiting ./rate-limiting.go
RUN cd prober/ctlog && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/ctlog-sth ./ctlog-sth.go
RUN cd pager-duty && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/pager .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:627d6c5a23ad24e6bdff827f16c7b60e0289029b0c79e9f7ccd54ae3279fb45f

ARG TOOLS_DIR
COPY --from=go-builder ${TOOLS_DIR}/* /usr/local/bin/
