FROM ubuntu

ADD edgemesh-iptables.sh /usr/local/bin
RUN apt-get update && apt-get install -y iproute2 iptables

LABEL name="edgemesh iptables traffic hijack Image " \
		vendor="CentOS" \
		license="" \
		
ENTRYPOINT ["usr/local/bin/edgemesh-iptables.sh"]