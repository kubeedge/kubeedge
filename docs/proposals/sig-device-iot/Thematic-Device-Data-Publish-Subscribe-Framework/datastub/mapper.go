package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"

    pb "gitee.com/fudan-se/kubeedge/extensive/datastub/MQProductor/v1beta1"
)

// Mapper 配置
type PropertyConfig struct {
    Name       string
    PushMethod string // datastub_grpc | datastub_http | twin
    Qos        string // RT | BT
}

type MapperConfig struct {
    DeviceName string
    GRPCTarget string
    Properties []PropertyConfig
}

func main() {
    cfg := MapperConfig{
        DeviceName: "machine-1",
        GRPCTarget: "127.0.0.1:18080", // Producer gRPC 地址
        Properties: []PropertyConfig{
            {Name: "sensor/temperature", PushMethod: "datastub_grpc", Qos: "RT"},
        },
    }

    // 建立 gRPC 连接
    conn, err := grpc.Dial(cfg.GRPCTarget, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("failed to connect to producer: %v", err)
    }
    defer conn.Close()

    client := pb.NewDataStubServiceClient(conn)

    // 模拟定时采集温度数据并上报
    for {
        tempValue := "28.5" // 假设采集到的数据

        for _, prop := range cfg.Properties {
            if prop.PushMethod == "datastub_grpc" {
                // 调用 Producer 的 gRPC 接口
                _, err := client.PushDeviceData(context.Background(), &pb.PushDeviceDataRequest{
                    Device:   &pb.Device{Name: cfg.DeviceName},
                    Property: &pb.DeviceProperty{Name: prop.Name, Value: tempValue},
                    Qos:      prop.Qos,
                })
                if err != nil {
                    log.Printf("failed to push data to producer: %v", err)
                } else {
                    log.Printf("pushed %s=%s (qos=%s) to producer", prop.Name, tempValue, prop.Qos)
                }
            }
        }

        time.Sleep(5 * time.Second)
    }
}
