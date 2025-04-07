ARG TOOLS_DIR="/go/tools"

FROM golang:1.24.2-alpine@sha256:2d40d4fc278dad38be0777d5e2a88a2c6dee51b0b29c97a764fc6c6a11ca893c AS go-builder
ARG TOOLS_DIR

RUN mkdir ${TOOLS_DIR}

ADD prober prober/
ADD pager-duty pager-duty/

# the specific versions of these tools are in prober/hack/go.mod so that Dependabot can bump them for updates
RUN cd prober/hack && GOBIN=${TOOLS_DIR} go install -trimpath -ldflags="-w -s" tool

RUN cd prober && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/rate-limiting ./rate-limiting.go
RUN cd prober/ctlog && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/ctlog-sth ./ctlog-sth.go
RUN cd pager-duty && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/pager .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc

ARG TOOLS_DIR
COPY --from=go-builder ${TOOLS_DIR}/* /usr/local/bin/
