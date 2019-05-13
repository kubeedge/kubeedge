FROM ubuntu:16.04

CMD mkdir -p kubeedge

COPY . kubeedge/

WORKDIR kubeedge

ENTRYPOINT ["/kubeedge/main","-logtostderr=true"]