---
title: Implementing Mapper Framework in C language
status: implementable
authors:
  - "@ZhangTianxi05"
approvers:
  - "@WillardHu"
  - "@Shelley-BaoYue"
  - "@zhijiayang"
creation-date: 2025-07-04
last-updated: 2025-07-04
---

# Implementing Mapper Framework in C language
- [Edge Resource Upgrade Control](#edge-resource-upgrade-control)
  - [Motivation / Background](#motivation--background)
  - [Use cases](#use-cases)
  - [Proposal Design](#proposal-design)
    - [File Structure](#file-structure)
    - [Dependency](#dependency)
    - [Workflow Redesign](#workflow-redesign)
    - [Testing](#testing)

## Motivation / Background

KubeEdge's Mapper Framework provides a new Mapper auto generation framework that integrates DMI device management and data capabilities. At present, KubeEdge multilingual Mapper Framework has been implemented in Golang and Java versions. However, in the IoT field, most edge side device drivers are written in C language. Therefore, in this project, we hope to provide a C language implementation of Mapper Framework to provide users with a C language based device driver Mapper and improve user development efficiency.

## Use cases

- C language

Currently, the Mapper Framework has implementations in both Go and Java. However, many hardware devices—especially those used in the IoT and embedded domains—are primarily developed using the C programming language. As a result, there is a strong need for a C language version of the Mapper Framework. By providing a C-based implementation, we can enable seamless integration with existing device drivers and firmware, reduce the learning curve for developers who are already proficient in C, and improve the overall efficiency and flexibility of device development. This will make it significantly easier for organizations to adopt the Mapper Framework in environments where C is the dominant language, thereby broadening the framework’s applicability and accelerating the development of edge computing solutions.

## Proposal Design

### File Structure

This is the file structure I have designed:
```
mapper-framework-c/
├── CMakeLists.txt                # CMake build script (or Makefile)
├── README.md
├── LICENSE
├── config/
│   └── config.yaml               # Configuration file
├── include/                      # Public header files, exposed interfaces
│   ├── config.h                  # Configuration parsing declarations
│   ├── datamodel.h               # Data model structure declarations
│   ├── datamethod.h              # Data method structure declarations
│   ├── device.h                  # Device-related structures and interfaces
│   ├── devicetwin.h              # Twin-related structures and interfaces
│   ├── driver.h                  # Driver-related interfaces
│   ├── grpc_client.h             # gRPC client interface
│   ├── grpc_server.h             # gRPC server interface
│   ├── http_server.h             # HTTP server interface
│   ├── influxdb2.h               # InfluxDB2 related interface
│   ├── redis.h                   # Redis related interface
│   ├── mysql.h                   # MySQL related interface
│   ├── event.h                   # Event-related interface
│   ├── util.h                    # Utility function declarations
│   ├── log.h                     # Logging interface
│   └── const.h                   # Constant definitions
├── src/                          # Main source code implementation
│   ├── main.c                    # Program entry
│   ├── config.c
│   ├── datamodel.c
│   ├── datamethod.c
│   ├── device.c
│   ├── devicetwin.c
│   ├── driver.c
│   ├── grpc_client.c
│   ├── grpc_server.c
│   ├── http_server.c
│   ├── influxdb2.c
│   ├── redis.c
│   ├── mysql.c
│   ├── event.c
│   ├── util.c
│   ├── log.c
│   └── const.c
├── data/                         # Data publishing and database implementation layer
│   ├── dbmethod/
│   │   ├── influxdb2_client.h
│   │   ├── influxdb2_client.c
│   │   ├── redis_client.h
│   │   ├── redis_client.c
│   │   ├── mysql_client.h
│   │   ├── mysql_client.c
│   │   └── ...
│   ├── publish/
│   │   ├── mqtt_client.h
│   │   ├── mqtt_client.c
│   │   ├── http_client.h
│   │   ├── http_client.c
│   │   └── ...
│   └── stream/
│       ├── img.h
│       ├── img.c
│       ├── video.h
│       ├── video.c
│       ├── handler_stream.h
│       ├── handler_stream.c
│       ├── handler_nostream.h
│       ├── handler_nostream.c
│       └── ...
├── device/                       # Device layer implementation
│   ├── device.h
│   ├── device.c
│   ├── devicetwin.h
│   ├── devicetwin.c
│   └── ...
├── driver/                       # Device driver layer
│   ├── driver.h
│   ├── driver.c
│   ├── devicetype.h
│   ├── devicetype.c
│   └── ...
├── grpc/                         # gRPC communication
│   ├── grpc_client.h
│   ├── grpc_client.c
│   ├── grpc_server.h
│   ├── grpc_server.c
│   └── ...
├── http/                         # HTTP server
│   ├── http_server.h
│   ├── http_server.c
│   └── ...
├── proto/                        # proto files and generated C code
│   ├── api.proto
│   ├── api.pb-c.h
│   ├── api.pb-c.c
│   └── ...
├── hack/                         # Helper scripts and tools
│   ├── make-rules/
│   │   ├── build.sh
│   │   ├── generate.sh
│   │   ├── crossbuild.sh
│   │   └── ...
│   ├── lib/
│   │   ├── init.sh
│   │   ├── install.sh
│   │   ├── lint.sh
│   │   └── util.sh
│   └── ...
├── scripts/                      # Other build, test, and deployment scripts
│   ├── build.sh
│   ├── generate_proto.sh
│   └── ...
├── test/                         # Unit test code
│   ├── test_device.c
│   ├── test_driver.c
│   └── ...
├── thirdparty/                   # Third-party dependencies (e.g., protobuf-c, json-c, etc.)
│   └── ...
├── build/                        # Build output directory
└── docs/                         # Project documentation
    └── architecture.md
```

### Dependency

To migrate the mapper-framework from Go to C, we need to introduce some third-party C libraries, such as yaml, etc.
Below is a sample C implementation of the config module:

```c
#include "config.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <yaml.h>

Config *config_parse(const char *filename)
{
    FILE *fh = fopen(filename, "r");
    if (!fh)
        return NULL;

    Config *cfg = (Config *)calloc(1, sizeof(Config));
    if (!cfg)
    {
        fclose(fh);
        return NULL;
    }

    yaml_parser_t parser;
    yaml_token_t token;
    char key[128] = {0};
    int in_grpc_server = 0, in_common = 0;

    if (!yaml_parser_initialize(&parser))
    {
        fclose(fh);
        free(cfg);
        return NULL;
    }
    yaml_parser_set_input_file(&parser, fh);

    while (1)
    {
        yaml_parser_scan(&parser, &token);
        if (token.type == YAML_STREAM_END_TOKEN)
            break;

        if (token.type == YAML_KEY_TOKEN)
        {
            yaml_token_delete(&token);
            yaml_parser_scan(&parser, &token);
            if (token.type == YAML_SCALAR_TOKEN)
            {
                if (strcmp((char *)token.data.scalar.value, "grpc_server") == 0)
                {
                    in_grpc_server = 1;
                    in_common = 0;
                    key[0] = '\0';
                }
                else if (strcmp((char *)token.data.scalar.value, "common") == 0)
                {
                    in_grpc_server = 0;
                    in_common = 1;
                    key[0] = '\0';
                }
                else
                {
                    strncpy(key, (char *)token.data.scalar.value, sizeof(key) - 1);
                    key[sizeof(key) - 1] = '\0';
                }
            }
        }
        else if (token.type == YAML_VALUE_TOKEN)
        {
            yaml_token_delete(&token);
            yaml_parser_scan(&parser, &token);
            if (token.type == YAML_SCALAR_TOKEN)
            {
                if (in_grpc_server)
                {
                    if (strcmp(key, "socket_path") == 0)
                        strncpy(cfg->grpc_server.socket_path, (char *)token.data.scalar.value, sizeof(cfg->grpc_server.socket_path) - 1);
                }
                else if (in_common)
                {
                    if (strcmp(key, "name") == 0)
                        strncpy(cfg->common.name, (char *)token.data.scalar.value, sizeof(cfg->common.name) - 1);
                    else if (strcmp(key, "version") == 0)
                        strncpy(cfg->common.version, (char *)token.data.scalar.value, sizeof(cfg->common.version) - 1);
                    else if (strcmp(key, "api_version") == 0)
                        strncpy(cfg->common.api_version, (char *)token.data.scalar.value, sizeof(cfg->common.api_version) - 1);
                    else if (strcmp(key, "protocol") == 0)
                        strncpy(cfg->common.protocol, (char *)token.data.scalar.value, sizeof(cfg->common.protocol) - 1);
                    else if (strcmp(key, "address") == 0)
                        strncpy(cfg->common.address, (char *)token.data.scalar.value, sizeof(cfg->common.address) - 1);
                    else if (strcmp(key, "edgecore_sock") == 0)
                        strncpy(cfg->common.edgecore_sock, (char *)token.data.scalar.value, sizeof(cfg->common.edgecore_sock) - 1);
                    else if (strcmp(key, "http_port") == 0)
                        strncpy(cfg->common.http_port, (char *)token.data.scalar.value, sizeof(cfg->common.http_port) - 1);
                }
            }
        }
        yaml_token_delete(&token);
    }

    yaml_parser_delete(&parser);
    fclose(fh);
    return cfg;
}

void config_free(Config *cfg)
{
    if (cfg)
        free(cfg);
}
```

### Workflow Redesign

We need to redesign the build and deployment process to fit C language compilation and packaging workflows (such as Makefile, CMake, Dockerfile, etc.).

#### Language Migration (Go → C)

- Rewrite the core functionalities of the Go version (such as configuration parsing, device communication, protocol adaptation, data synchronization, etc.) in C.
- Replace Go ecosystem dependencies with their C language equivalents (for example, use libyaml for YAML parsing, protobuf-c for gRPC, libcurl for HTTP, and appropriate C libraries for database access).
- Redesign the build and deployment process to fit C language compilation and packaging workflows (such as Makefile, CMake, Dockerfile, etc.).
- Maintain consistent interfaces and features with the Go version to facilitate future maintenance and multi-language collaboration.

### Testing

- In the `test` folder, I have added test code for each designed module, such as `config_test.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include "config.h"

int main() {
    Config *cfg = config_parse("config/config.yaml");
    if (!cfg) {
        printf("Failed to parse config\n");
        return 1;
    }

    printf("GRPC socket_path: %s\n", cfg->grpc_server.socket_path);
    printf("Common name: %s\n", cfg->common.name);
    printf("Common version: %s\n", cfg->common.version);
    printf("Common api_version: %s\n", cfg->common.api_version);
    printf("Common protocol: %s\n", cfg->common.protocol);
    printf("Common address: %s\n", cfg->common.address);
    printf("Common edgecore_sock: %s\n", cfg->common.edgecore_sock);
    printf("Common http_port: %s\n", cfg->common.http_port);

    config_free(cfg);
    return 0;
}
```