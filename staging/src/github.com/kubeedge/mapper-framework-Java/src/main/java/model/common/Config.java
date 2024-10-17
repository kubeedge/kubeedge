package model.common;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

@Getter @Setter
public class Config {
    @JsonProperty("grpc_server")
    private GrpcServer grpcServer;

    @JsonProperty("common")
    private Common common;

    public Config(){}
    public Config(GrpcServer grpcServer, Common common) {
        this.grpcServer = grpcServer;
        this.common = common;
    }

    @Getter @Setter
    public static class Common {
        @JsonProperty("name")
        private String name = "";

        @JsonProperty("version")
        private String version = "";

        @JsonProperty("api_version")
        private String apiVersion = "";

        @JsonProperty("protocol")
        private String protocol = "";

        @JsonProperty("address")
        private String address = "";

        @JsonProperty("edgecore_sock")
        private String edgeCoreSock = "";

        @JsonProperty("http_port")
        private String httpPort = "";

        public Common(){}
        public Common(String name, String version, String apiVersion, String protocol, String address, String edgeCoreSock, String httpPort) {
            this.name = name;
            this.version = version;
            this.apiVersion = apiVersion;
            this.protocol = protocol;
            this.address = address;
            this.edgeCoreSock = edgeCoreSock;
            this.httpPort = httpPort;
        }
    }

    @Getter @Setter
    public static class GrpcServer {
        @JsonProperty("socket_path")
        private String socketPath = "";

        public GrpcServer(){}
        public GrpcServer(String socketPath) {
            this.socketPath = socketPath;
        }
    }
}


