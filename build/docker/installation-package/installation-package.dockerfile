FROM kubeedge/build-tools:1.17.13-ke1 as builder
WORKDIR /work
ADD . .
RUN mkdir -p bin && \
    make WHAT=edgecore BUILD_WITH_CONTAINER=false && cp _output/local/bin/edgecore bin/edgecore && \
    make WHAT=keadm BUILD_WITH_CONTAINER=false && cp _output/local/bin/keadm bin/keadm

FROM ubuntu:18.04
COPY --from=builder /work/_output/local/bin/edgecore /usr/local/bin/edgecore
COPY --from=builder /work/_output/local/bin/keadm /usr/local/bin/keadm

WORKDIR /etc/kubeedge
# Custom image can add more content here.
# e.g. config
