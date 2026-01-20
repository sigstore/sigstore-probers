ARG TOOLS_DIR="/go/tools"

FROM golang:1.25.6-alpine@sha256:d9b2e14101f27ec8d09674cd01186798d227bb0daec90e032aeb1cd22ac0f029 AS go-builder
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
RUN cd prober/prober && go build -trimpath -ldflags="-w -s" -o ${TOOLS_DIR}/prober .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:f9f84bd968430d7d35e8e6d55c40efb0b980829ec42920a49e60e65eac0d83fc

ARG TOOLS_DIR
COPY --from=go-builder ${TOOLS_DIR}/* /usr/local/bin/
