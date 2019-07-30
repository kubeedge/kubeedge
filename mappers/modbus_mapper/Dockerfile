FROM alpine:latest

COPY src/ /opt/src
COPY conf/ /opt/src/conf
COPY scripts/ /opt/scripts

RUN chmod +x /opt/scripts/start_modbusmapper.sh
RUN apk add --update nodejs

CMD sh /opt/scripts/start_modbusmapper.sh
