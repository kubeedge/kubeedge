package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "sync"

    zmq "github.com/pebbe/zmq4"
)

type Config struct {
    CloudRegistryURL string `json:"cloud_registry_url"`
}

type ProducerInfo struct {
    ID      string   `json:"id"`
    Address string   `json:"address"`
    Topics  []string `json:"topics"`
}

var (
    config       Config
    subscriber   *zmq.Socket
    dataCache    sync.Map                    // 用于存储设备数据
    channelMap   = make(map[string]chan []byte) // 用于存储每个请求的channel
    channelLock  sync.Mutex                  // 锁来保护 channelMap 的访问
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

// 处理订阅请求
func handleSubscribeRequest(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    var request struct {
        Topic string `json:"topic"`
    }
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        log.Printf("Failed to decode JSON data: %v", err)
        http.Error(w, "Failed to decode JSON data", http.StatusBadRequest)
        return
    }
    log.Printf("Received subscribe request: %+v", request)

    // 查找对应 topic 的生产者
    producer, err := getProducerForTopic(request.Topic)
    if err != nil {
        http.Error(w, "Failed to get producer for the requested topic", http.StatusInternalServerError)
        return
    }

    // 创建一个 channel 用于接收订阅到的数据
    dataChannel := make(chan []byte)
    channelLock.Lock()
    channelMap[request.Topic] = dataChannel
    channelLock.Unlock()

    // 订阅生产者的 ZeroMQ 发布
    if err := subscribeToProducer(producer, request.Topic); err != nil {
        http.Error(w, "Failed to subscribe to producer", http.StatusInternalServerError)
        return
    }

    // 持续向调用者返回订阅到的数据
    w.Header().Set("Content-Type", "application/json")
    for {
        select {
        case data := <-dataChannel:
            _, err := w.Write(data)
            if err != nil {
                log.Printf("Failed to write data to response: %v", err)
                return
            }
            if f, ok := w.(http.Flusher); ok {
                f.Flush() // 刷新以确保数据发送
            }
        case <-r.Context().Done():
            log.Printf("Client canceled the request for topic: %s", request.Topic)
            return
        }
    }
}

// 根据 topic 从云端注册表中查找生产者
func getProducerForTopic(topic string) (ProducerInfo, error) {
    url := config.CloudRegistryURL + "/get_producer?topic=" + topic
    resp, err := http.Get(url)
    if err != nil {
        return ProducerInfo{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return ProducerInfo{}, fmt.Errorf("failed to get producer, status code: %d", resp.StatusCode)
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return ProducerInfo{}, err
    }

    var producer ProducerInfo
    err = json.Unmarshal(body, &producer)
    if err != nil {
        return ProducerInfo{}, err
    }

    return producer, nil
}

// 订阅到生产者的 ZeroMQ 发布
func subscribeToProducer(producer ProducerInfo, topic string) error {
    var err error
    if subscriber == nil {
        subscriber, err = zmq.NewSocket(zmq.SUB)
        if err != nil {
            return fmt.Errorf("Failed to create ZeroMQ SUB socket: %v", err)
        }
    }

    // 连接到生产者地址
    err = subscriber.Connect(producer.Address)
    if err != nil {
        return fmt.Errorf("Failed to connect to producer %s: %v", producer.Address, err)
    }

    // 订阅指定的主题
    err = subscriber.SetSubscribe(topic)
    if err != nil {
        return fmt.Errorf("Failed to subscribe to topic %s: %v", topic, err)
    }

    log.Printf("Subscribed to topic: %s at address: %s", topic, producer.Address)

    // 在后台接收数据并通过 channel 返回
    go func() {
        for {
            // 接收 topic + message
            receivedMessage, err := subscriber.Recv(0)
            if err != nil {
                log.Printf("Failed to receive message: %v", err)
                continue
            }

            // 分割消息为 topic 和内容
            parts := strings.SplitN(receivedMessage, " ", 2)
            if len(parts) != 2 {
                log.Printf("Invalid message format, expecting topic and message")
                continue
            }

            receivedTopic := parts[0]
            message := parts[1]

            // 检查是否是订阅的 topic
            if receivedTopic == topic {
                log.Printf("Received data on topic %s: %s", receivedTopic, message)

                response := map[string]string{
                    "topic": receivedTopic,
                    "data":  message,
                }

                responseData, err := json.Marshal(response)
                if err != nil {
                    log.Printf("Failed to encode JSON data: %v", err)
                    continue
                }

                channelLock.Lock()
                if channel, exists := channelMap[topic]; exists {
                    channel <- responseData
                }
                channelLock.Unlock()
            }
        }
    }()
    return nil
}

func main() {
    // 加载配置文件
    var err error
    config, err = loadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    log.Println("Configuration loaded successfully")

    // 处理订阅请求的接口
    http.HandleFunc("/subscribe", handleSubscribeRequest)

    log.Println("Starting consumer server at :8090")
    http.ListenAndServe(":8090", nil)
}
