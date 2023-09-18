# Extract graphviz and dependencies
FROM golang:1.19.1-bullseye AS deb_extractor
RUN cd /tmp && \
    apt-get update && apt-get download \
        graphviz libgvc6 libcgraph6 libltdl7 libxdot4 libcdt5 libpathplan4 libexpat1 zlib1g && \
    mkdir /dpkg && \
    for deb in *.deb; do dpkg --extract $deb /dpkg || exit 10; done

FROM golang:1.19.1-bullseye AS builder
COPY . /go/src/pprofweb/
WORKDIR /go/src/pprofweb
RUN go build -o server ./webserver

FROM gcr.io/distroless/base-debian11:latest AS run
COPY --from=builder /go/src/pprofweb/server /pprofweb
COPY --from=deb_extractor /dpkg /
# Configure dot plugins
RUN ["dot", "-c"]

WORKDIR /
EXPOSE 7443
ENTRYPOINT ["/pprofweb"]
