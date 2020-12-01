FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY ./modbus kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge

ENTRYPOINT ["/kubeedge/modbus", "--v", "5"]
