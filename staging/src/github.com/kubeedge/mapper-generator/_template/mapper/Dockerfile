FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY ./bin/Template kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge

CMD ./Template
