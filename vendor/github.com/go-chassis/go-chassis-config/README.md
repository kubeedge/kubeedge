### go-chassis-config
[![Build Status](https://travis-ci.org/go-chassis/go-chassis-config.svg?branch=master)](https://travis-ci.org/go-chassis/go-chassis-config)  
go-chassis-config is able to pull configs from heterogeneous distributed configuration 
management service.
it is decoupled with go chassis. you can use it directly without go chassis.

Supported distributed configuration management service:

| name       | import                                         |description    |
|----------|----------|:-------------:|
|config_center |github.com/go-chassis/go-chassis-config/configcenter |huawei cloud CSE config center https://www.huaweicloud.com/product/cse.html |
|apollo(not longer under maintenance)      |github.com/go-chassis/go-chassis-config/apollo       |ctrip apollo https://github.com/ctripcorp/apollo |

# Example
Get a client of config center

1. import the config client you want to use 
```go
import _ "github.com/go-chassis/go-chassis-config/configcenter"
```

2. Create a client 
```go
c, err := ccclient.NewClient("config_center", ccclient.Options{
		ServerURI: "http://127.0.0.1:30200",
	})
````

# Use huawei cloud 
```go
import (
	"github.com/huaweicse/auth"
	"github.com/go-chassis/foundation/httpclient"
	_ "github.com/go-chassis/go-chassis-config/configcenter"
)

func main() {
	var err error
	httpclient.SignRequest,err =auth.GetShaAKSKSignFunc("your ak", "your sk", "")
	if err!=nil{
        //handle err
	}
	ccclient.NewClient("config_center",ccclient.Options{
		ServerURI:"the address of CSE endpoint",
	})
}

```