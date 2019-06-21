FROM centos:latest

ADD edgemesh-iptables.sh /usr/local/bin

RUN yum -y update && yum install -y iproute iptables
		
ENTRYPOINT ["/usr/local/bin/edgemesh-iptables.sh"]
