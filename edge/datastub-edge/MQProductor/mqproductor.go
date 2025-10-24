package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "time"
    "net/http"
    "os"
    "strconv"
    "sync"
    "bytes"

    zmq "github.com/pebbe/zmq4"
    pb "gitee.com/fudan-se/kubeedge/extensive/datastub/MQProductor/v1beta1"
    "google.golang.org/grpc"
)

type Config struct {
    CloudRegistryURL string `json:"cloud_registry_url"`
    IngestGRPCAddr   string `json:"ingest_grpc_addr"`
    IngestHTTPAddr   string `json:"ingest_http_addr"`
}

type ProducerInfo struct {
    ID      string   `json:"id"`
    Address string   `json:"address"`
    Topics  []string `json:"topics"`
}

var (
    config            Config
    publisherMap      = make(map[string]struct{ publisher *zmq.Socket; port int }) // 每个 topic 对应的 ZeroMQ 发布者和端口信息
    devicePortMapping = make(map[string]int)                                        // 设备名与端口映射
    portCounter       = 5556                                                        // 起始端口号，每个设备使用一个独立端口
    portMutex         sync.Mutex                                                     // 用于获取端口时的锁
    registryMutex     sync.Mutex                                                     // 用于注册表访问的锁
    producerRegistry  = make(map[string][]ProducerInfo)                             // 注册表，按 topic 维护的生产者信息

    rtChan = make(chan publishTask, 1000) // 高优先级，实时
    btChan = make(chan publishTask, 1000) // 低优先级，批量

)

// 加载配置文件
func loadConfig() (Config, error) {
    file, err := os.Open("config.json")
    if err != nil {
        return Config{}, err
    }
    defer file.Close()

    var config Config
    err = json.NewDecoder(file).Decode(&config)
    return config, err
}

// 获取一个新的端口号
func getNewPort() int {
    portMutex.Lock()
    defer portMutex.Unlock()
    port := portCounter
    portCounter++
    return port
}

// 注册生产者到云端注册表
func registerProducer(deviceName, topic string, port int) {
    producerInfo := ProducerInfo{
        ID:      fmt.Sprintf("%s:%s", deviceName, topic), // 唯一标识 deviceName:topic
        Address: "tcp://" + getLocalIP() + ":" + strconv.Itoa(port),
        Topics:  []string{topic},
    }

    jsonData, err := json.Marshal(producerInfo)
    if err != nil {
        log.Fatalf("Failed to marshal producer data: %v", err)
    }

    resp, err := http.Post(config.CloudRegistryURL+"/register", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        log.Fatalf("Failed to register producer: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Fatalf("Failed to register producer, status code: %d", resp.StatusCode)
    }

    log.Printf("Producer registered successfully: %s for topic: %s on port %d", deviceName, topic, port)
}

// 检查设备是否已经注册
func isDeviceTopicRegistered(deviceName, topic string) bool {
    registryMutex.Lock()
    defer registryMutex.Unlock()

    producers, exists := producerRegistry[topic]
    if !exists {
        return false
    }

    for _, producer := range producers {
        if producer.ID == fmt.Sprintf("%s:%s", deviceName, topic) {
            return true
        }
    }

    return false
}

// 启动 gRPC 服务器来接收设备数据
func startGRPCServer() {
    // socketPath := "/Users/zbx/Desktop/KubeEdge/data/ds.sock"
    // if _, err := os.Stat(socketPath); err == nil {
    //     os.Remove(socketPath)
    // }

    // listener, err := net.Listen("unix", socketPath) // 使用 Unix Socket 监听
    // if err != nil {
    //     log.Fatalf("Failed to start gRPC server: %v", err)
    // }
    // defer listener.Close()

    // // 设置文件权限，确保客户端能访问
    // if err := os.Chmod(socketPath, 0777); err != nil {
    //     log.Fatalf("Failed to chmod on %s: %v", socketPath, err)
    // }
    // grpcServer := grpc.NewServer()
    // pb.RegisterDataStubServiceServer(grpcServer, &Server{})

    // log.Printf("gRPC server is listening on Unix Socket %s", socketPath)

    // if err := grpcServer.Serve(listener); err != nil {
    //     log.Fatalf("Failed to serve gRPC server: %v", err)
    // }

    listener, err := net.Listen("tcp", config.IngestGRPCAddr)
    if err != nil {
        log.Fatalf("Failed to start gRPC server: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterDataStubServiceServer(grpcServer, &Server{})

    log.Printf("gRPC server is listening on %s", config.IngestGRPCAddr)

    if err := grpcServer.Serve(listener); err != nil {
        log.Fatalf("Failed to serve gRPC server: %v", err)
    }

}

//http几万块
// func startHTTPServer() {
//     http.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
//         var req pb.PushDeviceDataRequest
//         if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//             http.Error(w, err.Error(), http.StatusBadRequest)
//             return
//         }
//         _, err := (&Server{}).PushDeviceData(context.Background(), &req)
//         if err != nil {
//             http.Error(w, err.Error(), http.StatusInternalServerError)
//             return
//         }
//         w.WriteHeader(http.StatusOK)
//     })

//     log.Printf("HTTP ingest server is listening on %s", config.IngestHTTPAddr)
//     if err := http.ListenAndServe(config.IngestHTTPAddr, nil); err != nil {
//         log.Fatalf("Failed to start HTTP server: %v", err)
//     }
// }


// Server 是实现 gRPC 服务接口的结构体
type Server struct {
    pb.UnimplementedDataStubServiceServer
}

// 定义一个任务结构体
type publishTask struct {
    topic     string
    device    string
    payload   []byte
    qos       string
}

// 实现 gRPC 服务的方法
func (s *Server) PushDeviceData(ctx context.Context, req *pb.PushDeviceDataRequest) (*pb.PushDeviceDataResponse, error) {
    log.Printf("Received data from device: %s", req.Device.Name)
    log.Printf("Device property: %s, value: %s, qos: %s", req.Property.Name, req.Property.Value, req.Qos)

    topic := req.Property.Name
    deviceName := req.Device.Name

    // 检查是否已注册 deviceName:topic，如果没有则分配新端口并注册
    if !isDeviceTopicRegistered(deviceName, topic) {
        port := getNewPort()

        // 创建 ZeroMQ 发布者
        publisher, err := zmq.NewSocket(zmq.PUB)
        if err != nil {
            log.Printf("Failed to create ZeroMQ PUB socket for topic %s: %v", topic, err)
            return nil, err
        }

        err = publisher.Bind(fmt.Sprintf("tcp://*:%d", port))
        if err != nil {
            log.Printf("Failed to bind ZeroMQ publisher socket for topic %s: %v", topic, err)
            return nil, err
        }
        log.Printf("ZeroMQ PUB socket for topic %s created and bound to address tcp://*:%d", topic, port)

        publisherMap[fmt.Sprintf("%s:%s", deviceName, topic)] = struct {
            publisher *zmq.Socket
            port      int
        }{publisher: publisher, port: port}

        // 注册设备到云端
        registerProducer(deviceName, topic, port)

        // 更新生产者注册表
        registryMutex.Lock()
        producerRegistry[topic] = append(producerRegistry[topic], ProducerInfo{
            ID:      fmt.Sprintf("%s:%s", deviceName, topic),
            Address: "tcp://" + getLocalIP() + ":" + strconv.Itoa(port),
            Topics:  []string{topic},
        })
        registryMutex.Unlock()
    }

    // 发布到 ZeroMQ
    payload, err := json.Marshal(req)
    if err != nil {
        log.Printf("Failed to marshal message: %v", err)
        return nil, err
    }

    // key := fmt.Sprintf("%s:%s", deviceName, topic)
    // publisher := publisherMap[key].publisher
    // _, err = publisher.Send(topic+" "+string(payload), 0)
    // if err != nil {
    //     log.Printf("Failed to publish message from topic %s: %v", topic, err)
    //     return nil, err
    // }

    task := publishTask{
        topic:   topic,
        device:  deviceName,
        payload: payload,
        qos:     req.Qos,
    }
    if req.Qos == "RT" {
        rtChan <- task
    } else {
        btChan <- task
    }

    return &pb.PushDeviceDataResponse{}, nil
}

func startDispatcher() {
    go func() {
        btBuffer := make([]publishTask, 0, 100) // 缓冲区
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case task := <-rtChan: // 高优先级立即发
                key := fmt.Sprintf("%s:%s", task.device, task.topic)
                publisher := publisherMap[key].publisher
                publisher.Send(task.topic+" "+string(task.payload), 0)

            case task := <-btChan: // BT 先存到缓冲区
                btBuffer = append(btBuffer, task)

            case <-ticker.C: // 定时批量发送
                if len(btBuffer) > 0 {
                    for _, task := range btBuffer {
                        key := fmt.Sprintf("%s:%s", task.device, task.topic)
                        publisher := publisherMap[key].publisher
                        publisher.Send(task.topic+" "+string(task.payload), 0)
                    }
                    log.Printf("Flushed %d BT messages", len(btBuffer))
                    btBuffer = btBuffer[:0] // 清空缓冲
                }
            }
        }
    }()
}



// 获取本地 IP 地址
func getLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        log.Fatalf("Failed to get local IP: %v", err)
    }

    for _, addr := range addrs {
        if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
            if ipNet.IP.To4() != nil {
                return ipNet.IP.String()
            }
        }
    }

    log.Fatal("No valid local IP address found")
    return ""
}

func main() {
    // 加载配置文件
    var err error
    config, err = loadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    log.Println("Configuration loaded successfully")

    //启动调度器
    go startDispatcher()

    // 启动 gRPC ingest
    if config.IngestGRPCAddr != "" {
        go startGRPCServer()
    }

    // 启动 HTTP ingest
    // if config.IngestHTTPAddr != "" {
    //     go startHTTPServer()
    // }

    // 阻塞主线程，保持程序运行
    select {}
}
