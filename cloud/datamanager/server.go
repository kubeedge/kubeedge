package main

import (
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "sync"
    "time"
)

type ProducerInfo struct {
    ID      string   `json:"id"`
    Address string   `json:"address"`
    Topics  []string `json:"topics"`
}

var (
    producerRegistry = make(map[string][]ProducerInfo) // 按 topic 维护的生产者注册表
    registryMutex    sync.Mutex
)

// 注册 Producer
func registerProducer(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    var producer ProducerInfo
    err := json.NewDecoder(r.Body).Decode(&producer)
    if err != nil {
        http.Error(w, "Failed to decode JSON data", http.StatusBadRequest)
        return
    }

    registryMutex.Lock()
    defer registryMutex.Unlock()
    for _, topic := range producer.Topics {
        producerRegistry[topic] = append(producerRegistry[topic], producer)
    }

    w.WriteHeader(http.StatusOK)
    log.Printf("Producer registered successfully: %s", producer.ID)
}

// 获取 Producer 信息
func getProducerForTopic(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    topic := r.URL.Query().Get("topic")
    if topic == "" {
        http.Error(w, "Missing topic query parameter", http.StatusBadRequest)
        return
    }

    registryMutex.Lock()
    defer registryMutex.Unlock()

    producers, exists := producerRegistry[topic]
    if !exists || len(producers) == 0 {
        http.Error(w, "No producer available for the requested topic", http.StatusNotFound)
        return
    }

    // 随机选择一个生产者地址返回给消费者
    rand.Seed(time.Now().UnixNano())
    selectedProducer := producers[rand.Intn(len(producers))]

    response, err := json.Marshal(selectedProducer)
    if err != nil {
        http.Error(w, "Failed to encode JSON data", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(response)
}

// 获取所有注册表中的生产者信息
func getAllProducers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    registryMutex.Lock()
    defer registryMutex.Unlock()

    response, err := json.Marshal(producerRegistry)
    if err != nil {
        http.Error(w, "Failed to encode JSON data", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(response)
}

func main() {
    http.HandleFunc("/register", registerProducer)
    http.HandleFunc("/get_producer", getProducerForTopic)
    http.HandleFunc("/registry", getAllProducers) // 新增查看注册表的端点

    log.Println("Starting registration server at :8080")
    http.ListenAndServe(":8080", nil)
}



// curl -N -X POST http://localhost:8090/subscribe \
//   -H "Content-Type: application/json" \
//   -d '{"topic":"device/123/temperature"}'