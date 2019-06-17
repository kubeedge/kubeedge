FROM centos:latest

ADD edgemesh-iptables.sh /usr/local/bin

RUN yum update &&yum install -y iproute iptables

LABEL name="edgemesh iptables traffic hijack Image " \
		vendor="CentOS" \
		license="" \
		
ENTRYPOINT ["usr/local/bin/edgemesh-iptables.sh"]