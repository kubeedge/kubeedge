#!/usr/bin/bash

function usage() {
	echo 'this is the edgemesh-iptables usage'
	echo "${0} -p PROXY_PORT [-i HIJACK_IP] [-t HIJACK_PORT] [-b EXCLUDE_IP] [-c EXCLUDE_PORT] [-h]"
	echo ''
	echo '  -p: Specify the edgemesh port to which all TCP traffic from the Pod will be redirectd to. (default 10001)'
	echo '  -i: Comma separated list of outbound IP for which tarffic is to be redirectd to edgemesh. The'
	echo '      wildcard character "*" can be used to configure redirection for all IPs. (default "*")'
	echo '  -t: Comma separated list of outbound Port for which tarffic is to be redirectd to edgemesh. The'
	echo '      wildcard character "*" can be used to configure redirection for all Ports. (default "*")'
	echo '  -b: Comma separated list of outbound IP range in CIDR to be excluded from redirection to edgemesh.'
	echo '      The Empty character "" can be used to configure redirection for all IPs. (default "")'
	echo '  -c: Comma separated list of outbound Port to be excluded from redirection to edgemesh. The'
	echo '      Empty character "" can be used to configure redirection for all Ports. (default "")'
	echo '  -h: for some help'
}

# network namespace 
NETMODE=

# get the container network mode 
function getContainerNetMode() {
	if ip link |grep docker0 > /dev/null; then
		echo 'this is the host mode,share with net namespace with host'
		NETMODE='HOST'
	else
		echo 'this is the ohter container net mode(none,bridge),independent of the host net namespace'
		NETMODE='OTHER'
	fi
}

# judge if agrument is a valid ip address
function isValidIP() {
	if isIPv4 "${1}"; then
		true
	elif isIPv6 "${1}"; then
		true
	else
		false
	fi		
}

function isIPv4() {
	local ipv4matchString="^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$"
	if [[ ${1} =~ ${ipv4matchString} ]]; then
		true
	else
		false
	fi
}

function isIPv6() {
	# TODO
	false
}

function hostNetMode() {
	echo 'this func used for host net mode'
	echo 'TODO'
}

function bridgeNetMode() {
	echo 'this func used for bridge net mode'
	# get default route
	default_route=$(ip route show |grep default |awk '{print $3}')
	
	#clear EDGEMESH chain and rule,if exist
	iptables -t nat -D OUTPUT -p tcp -j EDGEMESH_OUTBOUND 2>/dev/null
	iptables -t nat -D OUTPUT -p udp --dport "53" -j EDGEMESH_OUTBOUND_DNS 2>/dev/null
	iptables -t nat -F EDGEMESH_OUTBOUND 2>/dev/null
	iptables -t nat -X EDGEMESH_OUTBOUND 2>/dev/null
	
	iptables -t nat -F EDGEMESH_OUTBOUND_REDIRECT 2>/dev/null
	iptables -t nat -X EDGEMESH_OUTBOUND_REDIRECT 2>/dev/null
	
	iptables -t nat -F EDGEMESH_OUTBOUND_DNS 2>/dev/null
	iptables -t nat -X EDGEMESH_OUTBOUND_DNS 2>/dev/null
	
	# make chain for edgemesh hijacking
	iptables -t nat -N EDGEMESH_OUTBOUND_REDIRECT
	iptables -t nat -A EDGEMESH_OUTBOUND_REDIRECT -p tcp -j DNAT --to-destination "${default_route}:${EDGEMESH_PROXY_PORT}"
	iptables -t nat -N EDGEMESH_OUTBOUND
	iptables -t nat -A OUTPUT -p tcp -j EDGEMESH_OUTBOUND
	
	# support dns use udp for dest port 53
	iptables -t nat -N EDGEMESH_OUTBOUND_DNS
	iptables -t nat -A EDGEMESH_OUTBOUND_DNS -j DNAT --to-destination "${default_route}"
	iptables -t nat -A OUTPUT -p udp --dport "53" -j EDGEMESH_OUTBOUND_DNS
	
	# excluded traffic for some port incloude some special port,such as 22
	iptables -t nat -A EDGEMESH_OUTBOUND -p tcp --dport "22" -j RETURN
	if [ -n "${EDGEMESH_EXCLUDE_PORT}" ]; then 
		for port in "${port_exclude_list[@]}"; do 
			iptables -t nat -A EDGEMESH_OUTBOUND -p tcp --dport "${port}" -j RETURN
		done
	fi
	# excluded traffic for some ips
	if [ ${#ipv4_exclude_list[@]} -gt 0 ]; then
		for ip in "${ipv4_exclude_list[@]}"; do
			iptables -t nat -A EDGEMESH_OUTBOUND -d "${ip}" -j RETURN
		done
	fi
	
	# Redirect app callback to itself via Servie IP （default not redirectd）
	get_local_IP=$(ip addr |grep inet|grep -v inet6|awk '{print $2}'|tr -d "addr:")
	
	for LOCAL_IP in $get_local_IP; do
		ele=${LOCAL_IP%$splt}
		echo "LOCAL_IP: $LOCAL_IP , $ele"
		if isIPv4 $ele; then
			iptables -t nat -A EDGEMESH_OUTBOUND -o lo ! -d "${LOCAL_IP}" -j EDGEMESH_OUTBOUND_REDIRECT
		fi
	done
	# loopback traffic
	iptables -t nat -A EDGEMESH_OUTBOUND -d 127.0.0.1/32 -j RETURN
	
	# hijacking
	if [ ${#ipv4_include_list[@]} -gt 0 ]; then
		# include Ips and ports are *
		if [[ "${ipv4_include_list}" == "*" && "${EDGEMESH_HIJACK_PORT}" == "*" ]]; then
			iptables -t nat -A EDGEMESH_OUTBOUND -p tcp -j EDGEMESH_OUTBOUND_REDIRECT
		else
			if [ "${ipv4_include_list}" != "*" ]; then
				for ip in "${ipv4_include_list[@]}"; do
					iptables -t nat -A EDGEMESH_OUTBOUND -p tcp -d "${ip}" -j EDGEMESH_OUTBOUND_REDIRECT
				done
			fi
			if [ "${EDGEMESH_HIJACK_PORT}" != "*" ]; then
				for port in "${port_include_list[@]}"; do 
					iptables -t nat -A EDGEMESH_OUTBOUND -p tcp --dport "${port}" -j EDGEMESH_OUTBOUND_REDIRECT
				done
			fi
			
			iptables -t nat -A EDGEMESH_OUTBOUND -j RETURN
		fi
	fi
}

# variable
ipv4_exclude_list=()
ipv4_include_list=()
ipv6_exclude_list=()
ipv6_exclude_list=()
port_exclude_list=()
port_include_list=()

splt='/*'
EDGEMESH_PROXY_PORT=${PROXY_PORT-10001}  # default PROXY_PORT 10001
EDGEMESH_HIJACK_IP=${HIJACK_IP-"*"}
EDGEMESH_HIJACK_PORT=${HIJACK_PORT-"*"}
EDGEMESH_EXCLUDE_IP=${EXCLUDE_IP-}
EDGEMESH_EXCLUDE_PORT=${EXCLUDE_PORT-}

function main() {
	getContainerNetMode
	
	while getopts ":p:i:t:b:c:h" opt; do
		case ${opt} in
			p)
				EDGEMESH_PROXY_PORT=${OPTARG}
				;;
			i)
				EDGEMESH_HIJACK_IP=${OPTARG}
				;;
			t) 
				EDGEMESH_HIJACK_PORT=${OPTARG}
				;;
			b)
				EDGEMESH_EXCLUDE_IP=${OPTARG}
				;;
			c) 
				EDGEMESH_EXCLUDE_PORT=${OPTARG}
				;;
			h)
				usage
				exit 0
				;;
			?)
				echo "Invalid option: -$OPTARG" >&2
				usage
				exit 1
				;;
		esac
	done
	
	echo "EdgeMesh iptables configration:"
	echo "====================================="
	echo "Container Network mode is: ${NETMODE}"
	echo "Variables:"
	echo "EDGEMESH_PROXY_PORT=${EDGEMESH_PROXY_PORT-10001}"
	echo "EDGEMESH_HIJACK_IP=${EDGEMESH_HIJACK_IP-"*"}"
	echo "EDGEMESH_HIJACK_PORT=${EDGEMESH_HIJACK_PORT-"*"}"
	echo "EDGEMESH_EXCLUDE_IP=${EDGEMESH_EXCLUDE_IP-}"
	echo "EDGEMESH_EXCLUDE_PORT=${EDGEMESH_EXCLUDE_PORT-}"
	
	# parse parameter
	IFS=',' read -ra EXCLUDE_IP <<< "${EDGEMESH_EXCLUDE_IP}"
	IFS=',' read -ra INCLUDE_IP <<< "${EDGEMESH_HIJACK_IP}"
	# echo "EXCLUDE_IP: ${EXCLUDE_IP}"
	for range in "${EXCLUDE_IP[@]}"; do
		r=${range%$splt}
		if isValidIP "$r"; then
			if isIPv4 "$r"; then
				ipv4_exclude_list+=("$range")
			elif isIPv6 "$r"; then
				ipv6_exclude_list+=("$range")
			fi
		fi
	done
	
	if [ "${EDGEMESH_HIJACK_IP}" == "*" ]; then
		ipv4_include_list=("*")
		ipv6_include_list=("*")
	else
		for range in "${INCLUDE_IP[@]}"; do
			r=${range%$splt}
			if isValidIP "$r";then
				if isIPv4 "$r"; then
					ipv4_include_list+=("$range")
				elif isIPv6 "$r"; then
					ipv6_include_list+=("$range")
				fi
			fi		
		done
	fi
	
	IFS=',' read -ra INCLUDE_PORT <<< "${EDGEMESH_HIJACK_PORT}"
	IFS=',' read -ra EXCLUDE_PORT <<< "${EDGEMESH_EXCLUDE_PORT}"
	if [ "${EDGEMESH_HIJACK_PORT}" != "*" ]; then
		for port in "${INCLUDE_PORT[@]}"; do
			port_include_list+=("$port")
		done
	fi
	
	if [ -n "${EDGEMESH_EXCLUDE_PORT}" ]; then 
		for port in "${EXCLUDE_PORT[@]}"; do 
			port_exclude_list+=("$port")
		done
	fi
	
	echo "ipv4_include_list : ${ipv4_include_list[@]}"
	echo "ipv4_exclude_list : ${ipv4_exclude_list[@]}"
	echo "port_include_list : ${port_include_list[@]}"
	echo "port_exclude_list : ${port_exclude_list[@]}"
	
	# bridge mode(port map) container network
	if [ "${NETMODE}" = "OTHER" ]; then	
		echo " ${NETMODE} iptables configration"
		bridgeNetMode
		# if set ipv6 option
		if false; then
			echo 'TODO'
		fi
	# host mode container network
	elif [ "${NETMODE}" = "HOST" ]; then
		#hostNetMode
		echo ${NETMODE}
		# if set ipv6 option
		if false; then
			echo 'TODO'
		fi
	else
		echo 'Dont support this container network '
	fi
}

# start to configure
main "${@}"
