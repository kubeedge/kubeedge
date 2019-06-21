FROM ubuntu

ADD edgemesh-iptables.sh /usr/local/bin
RUN apt-get update && apt-get install -y iproute2 iptables

ENTRYPOINT ["usr/local/bin/edgemesh-iptables.sh"]
