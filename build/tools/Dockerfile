FROM alpine:3.9

COPY certgen.sh /usr/local/bin/certgen

RUN apk --no-cache add coreutils openssl

VOLUME ["/etc/kubeedge/ca", "/etc/kubeedge/certs"]

ENTRYPOINT ["certgen"]
