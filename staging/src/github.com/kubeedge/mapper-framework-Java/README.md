# Mapper framework-Java
KubeEdge Device Mapper framework written in Java 11.

## Construction method
If you want to connect your edge device to KubeEdge, you can create and write customized mapper based on this project. Considering various devices, you need to redefined [CustomizedProtocolConfig](./src/main/java/model/CustomizedProtocolConfig.java) and [VisitorConfig](./src/main/java/model/VisitorConfig.java), then implement the [driver](./src/main/java/driver) layer. The main entry point of the project is [Main.java](./src/main/java/Main.java)
