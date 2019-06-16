## inbound NAT 模式下 --典型的nat 代理方式

# Use this chain also for redirecting inbound traffic to the common Envoy port
# when not using TPROXY.
iptables -t nat -N ISTIO_IN_REDIRECT ## 该子链实现入站流量 重定向
iptables -t nat -A ISTIO_IN_REDIRECT -p tcp -j REDIRECT --to-port "${INBOUND_CAPTURE_PORT}"


iptables -t nat -N ISTIO_INBOUND
iptables -t nat -A PREROUTING -p tcp -j ISTIO_INBOUND ## 在prerouting处 截获trafffic


if [ "${INBOUND_PORTS_INCLUDE}" == "*" ]; then
	iptables -t nat -A ISTIO_INBOUND -p tcp --dport 22 -j RETURN #端口22 直接放行 
	if [ -n "${INBOUND_PORTS_EXCLUDE}" ]; then ##对某些端口放行
      for port in ${INBOUND_PORTS_EXCLUDE}; do
        iptables -t nat -A ISTIO_INBOUND -p tcp --dport "${port}" -j RETURN
      done
    fi

	iptables -t nat -A ISTIO_INBOUND -p tcp -j ISTIO_IN_REDIRECT

else
	for port in ${INBOUND_PORTS_INCLUDE}; do
        iptables -t nat -A ISTIO_INBOUND -p tcp --dport "${port}" -j ISTIO_IN_REDIRECT
    done
fi


##mangle表的主要功能是根据规则修改数据包的一些标志位，以便其他规则或程序可以利用这种标志对数据包进行过滤或策略路由
##1.重定向一部分经过路由选择的流量到本地路由进程(类似NAT中的REDIRECT)
##2.使用非本地IP作为SOURCE IP初始化连接
##3.无需iptables参与，在非本地IP上起监听
##
## 
## inbond TPPROXY 模式下
# Use this chain also for redirecting inbound traffic to the common Envoy port
# when not using TPROXY.
iptables -t nat -N ISTIO_IN_REDIRECT ## 该子链实现入站流量 重定向
iptables -t nat -A ISTIO_IN_REDIRECT -p tcp -j REDIRECT --to-port "${INBOUND_CAPTURE_PORT}"

# When using TPROXY, create a new chain for routing all inbound traffic to
# Envoy. Any packet entering this chain gets marked with the ${INBOUND_TPROXY_MARK} mark,
# so that they get routed to the loopback interface in order to get redirected to Envoy.
# In the ISTIO_INBOUND chain, '-j ISTIO_DIVERT' reroutes to the loopback
# interface.
# Mark all inbound packets.
iptables -t mangle -N ISTIO_DIVERT
iptables -t mangle -A ISTIO_DIVERT -j MARK --set-mark "${INBOUND_TPROXY_MARK}" ##为经过 ISTIO_DIVERT 的trffic 打标记
iptables -t mangle -A ISTIO_DIVERT -j ACCEPT

# Route all packets marked in chain ISTIO_DIVERT using routing table ${INBOUND_TPROXY_ROUTE_TABLE}.
ip -f inet rule add fwmark "${INBOUND_TPROXY_MARK}" lookup "${INBOUND_TPROXY_ROUTE_TABLE}" #怎加一条路由策略
# In routing table ${INBOUND_TPROXY_ROUTE_TABLE}, create a single default rule to route all traffic to
# the loopback interface.
ip -f inet route add local default dev lo table "${INBOUND_TPROXY_ROUTE_TABLE}" || ip route show table all

# Create a new chain for redirecting inbound traffic to the common Envoy
# port.
# In the ISTIO_INBOUND chain, '-j RETURN' bypasses Envoy and
# '-j ISTIO_TPROXY' redirects to Envoy.
iptables -t mangle -N ISTIO_TPROXY
iptables -t mangle -A ISTIO_TPROXY ! -d 127.0.0.1/32 -p tcp -j TPROXY --tproxy-mark "${INBOUND_TPROXY_MARK}/0xffffffff" --on-port "${PROXY_PORT}"
#TPROXY target options:
#  --on-port port                    Redirect connection to port, or the original port if 0
#  --on-ip ip                        Optionally redirect to the given IP
#  --tproxy-mark value[/mask]        Mark packets with the given value/mask



iptables -t mangle -N ISTIO_INBOUND
iptables -t mangle -A PREROUTING -p tcp -j ISTIO_INBOUND

## 	以下和nat 模式相同，均是排除指定的端口，跳过
# Makes sure SSH is not redirected
if [ "${INBOUND_PORTS_INCLUDE}" == "*" ]; then
	iptables -t mangle -A ISTIO_INBOUND -p tcp --dport 22 -j RETURN
	# Apply any user-specified port exclusions.
	if [ -n "${INBOUND_PORTS_EXCLUDE}" ]; then
	  for port in ${INBOUND_PORTS_EXCLUDE}; do
		iptables -t mangle -A ISTIO_INBOUND -p tcp --dport "${port}" -j RETURN
	  done
	fi

	iptables -t mangle -A ISTIO_INBOUND -p tcp -m socket -j ISTIO_DIVERT || echo "No socket match support"
	# Otherwise, it's a new connection. Redirect it using TPROXY.
	iptables -t mangle -A ISTIO_INBOUND -p tcp -j ISTIO_TPROXY
else
	for port in ${INBOUND_PORTS_INCLUDE}; do
        iptables -t mangle -A ISTIO_INBOUND -p tcp --dport "${port}" -m socket -j ISTIO_DIVERT || echo "No socket match support"
        iptables -t mangle -A ISTIO_INBOUND -p tcp --dport "${port}" -m socket -j ISTIO_DIVERT || echo "No socket match support"
        iptables -t mangle -A ISTIO_INBOUND -p tcp --dport "${port}" -j ISTIO_TPROXY
	done

fi

	