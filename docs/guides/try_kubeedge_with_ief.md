# Try KubeEdge with HuaweiCloud (IEF) 

## [Intelligent EdgeFabric (IEF)](https://www.huaweicloud.com/product/ief.html)
**Note:** The HuaweiCloud IEF is only available in China now.

1. Create an account in [HuaweiCloud](https://www.huaweicloud.com).
2. Go to [IEF](https://www.huaweicloud.com/product/ief.html) and create an Edge node.
3. Download the node configuration file (<node_name>.tar.gz).
4. Run `cd $GOPATH/src/github.com/kubeedge/kubeedge/edge` to enter edge directory.
5. Run `bash -x hack/setup_for_IEF.sh /PATH/TO/<node_name>.tar.gz` to modify the configuration files in `conf/`.
