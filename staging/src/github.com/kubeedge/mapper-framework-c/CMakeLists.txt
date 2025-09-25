cmake_minimum_required(VERSION 3.10)
project(mapper_framework_c C CXX)

# Set C and C++ standards
set(CMAKE_C_STANDARD 99)
set(CMAKE_CXX_STANDARD 20)

# Required packages
find_package(PkgConfig REQUIRED)
pkg_check_modules(YAML REQUIRED yaml-0.1)
find_package(CURL REQUIRED)
find_package(Threads REQUIRED)

# Protobuf and gRPC
find_package(Protobuf CONFIG QUIET)
if(NOT Protobuf_FOUND)
  find_package(Protobuf REQUIRED)
endif()
find_package(gRPC CONFIG QUIET)

# FFmpeg (optional)
pkg_check_modules(FFMPEG
  libavformat
  libavcodec
  libavutil
  libswscale
)
if(NOT FFMPEG_FOUND)
  message(WARNING "FFmpeg libraries not found. Stream processing will be disabled.")
  set(STREAM_FOUND FALSE)
else()
  message(STATUS "Found FFmpeg libraries")
  set(STREAM_FOUND TRUE)
endif()

# hiredis library
find_path(HIREDIS_INCLUDE_DIR hiredis/hiredis.h
  PATHS /usr/include /usr/local/include)
find_library(HIREDIS_LIBRARY hiredis
  PATHS /usr/lib /usr/local/lib /usr/lib/x86_64-linux-gnu)
if(NOT HIREDIS_INCLUDE_DIR OR NOT HIREDIS_LIBRARY)
  message(FATAL_ERROR "hiredis library not found. Please install libhiredis-dev")
endif()

# TDengine (optional)
find_path(TAOS_INCLUDE_DIR taos.h
  PATHS /usr/include /usr/local/include /usr/local/taos/include)
find_library(TAOS_LIBRARY taos
  PATHS /usr/lib /usr/local/lib /usr/local/taos/driver /usr/lib/x86_64-linux-gnu)
if(NOT TAOS_INCLUDE_DIR OR NOT TAOS_LIBRARY)
  message(WARNING "TDengine (taos) library not found. TDengine support will be disabled.")
  set(TAOS_FOUND FALSE)
else()
  message(STATUS "Found TDengine: ${TAOS_LIBRARY}")
  set(TAOS_FOUND TRUE)
endif()

# cJSON and protobuf-c headers
include_directories(/usr/include/cjson)
include_directories(/usr/local/include/protobuf-c)
set(CJSON_LIBRARIES cjson)
set(PROTOBUF_C_LIBRARIES protobuf-c)

# Mosquitto (MQTT, optional)
find_path(MOSQUITTO_INCLUDE_DIR mosquitto.h
  PATHS /usr/include /usr/local/include)
find_library(MOSQUITTO_LIBRARY mosquitto
  PATHS /usr/lib /usr/local/lib /usr/lib/x86_64-linux-gnu)
if(MOSQUITTO_INCLUDE_DIR AND MOSQUITTO_LIBRARY)
  set(MOSQUITTO_FOUND TRUE)
  message(STATUS "Found Mosquitto: ${MOSQUITTO_LIBRARY}")
else()
  set(MOSQUITTO_FOUND FALSE)
  message(WARNING "Mosquitto not found. MQTT publish will be disabled.")
endif()

# Include directories
include_directories(
  ${YAML_INCLUDE_DIRS}
  ${PROTOBUF_C_INCLUDE_DIRS}
  ${HIREDIS_INCLUDE_DIR}
  ${CMAKE_CURRENT_SOURCE_DIR}
  ${CMAKE_CURRENT_SOURCE_DIR}/common
  ${CMAKE_CURRENT_SOURCE_DIR}/config
  ${CMAKE_CURRENT_SOURCE_DIR}/log
  ${CMAKE_CURRENT_SOURCE_DIR}/util/parse
  ${CMAKE_CURRENT_SOURCE_DIR}/dmi/v1beta1
  ${CMAKE_CURRENT_SOURCE_DIR}/google/protobuf
  ${CMAKE_CURRENT_SOURCE_DIR}/grpcclient
  ${CMAKE_CURRENT_SOURCE_DIR}/grpcserver
  ${CMAKE_CURRENT_SOURCE_DIR}/httpserver
  ${CMAKE_CURRENT_SOURCE_DIR}/device
  ${CMAKE_CURRENT_SOURCE_DIR}/data/dbmethod/mysql
  ${CMAKE_CURRENT_SOURCE_DIR}/data/dbmethod/influxdb2
  ${CMAKE_CURRENT_SOURCE_DIR}/data/dbmethod/redis
  ${CMAKE_CURRENT_SOURCE_DIR}/data/publish
  ${CMAKE_CURRENT_SOURCE_DIR}/data/publish/http
  ${CMAKE_CURRENT_SOURCE_DIR}/data/publish/otel
  ${CMAKE_CURRENT_SOURCE_DIR}/driver
  /usr/include
  /usr/local/include
)

if(MOSQUITTO_FOUND)
  include_directories(${MOSQUITTO_INCLUDE_DIR})
  include_directories(${CMAKE_CURRENT_SOURCE_DIR}/data/publish/mqtt)
endif()

if(TAOS_FOUND)
  include_directories(${TAOS_INCLUDE_DIR})
  include_directories(${CMAKE_CURRENT_SOURCE_DIR}/data/dbmethod/tdengine)
endif()

if(STREAM_FOUND)
  include_directories(${FFMPEG_INCLUDE_DIRS})
  include_directories(${CMAKE_CURRENT_SOURCE_DIR}/data/stream)
endif()

# absl libraries (used as fallback)
set(ABSL_LIBS
  absl_log_internal_check_op
  absl_log_internal_message
  absl_log_internal_format
  absl_log_internal_globals
  absl_log_internal_proto
  absl_log_internal_nullguard
  absl_synchronization
  absl_time
  absl_strings
  absl_status
  absl_base
  absl_cord
  absl_cord_internal
  absl_cordz_info
  absl_cordz_handle
  absl_cordz_functions
  absl_cordz_sample_token
  absl_hash
)

# Source files
set(COMMON_SOURCES
  # Core modules
  config/config.c
  log/log.c
  common/dataconverter.c
  common/datamodel.c
  common/event.c
  util/parse/grpc.c
  # Protobuf generated files
  dmi/v1beta1/api.pb-c.c
  dmi/v1beta1/api.pb.cc
  dmi/v1beta1/api.grpc.pb.cc
  google/protobuf/any.pb-c.c
  google/protobuf/wrappers.pb-c.c
  # Network services
  httpserver/httpserver.c
  grpcclient/register.cc
  grpcserver/server.cc
  # Device management
  device/device.c
  device/devicestatus.c
  device/devicetwin.c
  device/dev_panel.c
  # Driver framework
  driver/driver.c
  # Database clients
  data/dbmethod/mysql/mysql_client.c
  data/dbmethod/mysql/recorder.c
  data/dbmethod/influxdb2/influxdb2_client.c
  data/dbmethod/redis/redis_client.c
  # Publish modules (HTTP/OTEL, MQTT optional)
  data/publish/publisher.c
  data/publish/http/http_publisher.c
  data/publish/otel/otel_publisher.c
)

# Optional TDengine
if(TAOS_FOUND)
  list(APPEND COMMON_SOURCES
    data/dbmethod/tdengine/tdengine_client.c
  )
endif()

# Optional Stream
if(STREAM_FOUND)
  list(APPEND COMMON_SOURCES
    data/stream/stream.c
  )
endif()

# Optional MQTT publish
if(MOSQUITTO_FOUND)
  list(APPEND COMMON_SOURCES
    data/publish/mqtt/mqtt_publisher.c
  )
endif()

# Conditional compilation definitions
add_compile_definitions(
  $<$<BOOL:${TAOS_FOUND}>:TAOS_FOUND>
  $<$<BOOL:${STREAM_FOUND}>:ENABLE_STREAM>
  $<$<BOOL:${MOSQUITTO_FOUND}>:ENABLE_MQTT_PUBLISH>
)

# Main executable
add_executable(main
  main.c
  ${COMMON_SOURCES}
)

# gRPC/Protobuf libraries
set(GRPC_LINK_LIBS "")
set(PROTOBUF_LINK_LIB "")

if(TARGET gRPC::grpc++)
  list(APPEND GRPC_LINK_LIBS
    gRPC::grpc++
    gRPC::grpc
    gpr
    gRPC::grpc++_reflection
  )
else()
  list(APPEND GRPC_LINK_LIBS
    grpc++
    grpc
    gpr
    grpc++_reflection
  )
  list(APPEND GRPC_LINK_LIBS ${ABSL_LIBS})
endif()

if(TARGET protobuf::libprotobuf)
  list(APPEND PROTOBUF_LINK_LIB protobuf::libprotobuf)
else()
  list(APPEND PROTOBUF_LINK_LIB protobuf)
endif()

# Common libraries
set(COMMON_LIBRARIES
  ${CJSON_LIBRARIES}
  ${YAML_LIBRARIES}
  ${PROTOBUF_C_LIBRARIES}
  ${CURL_LIBRARIES}
  ${HIREDIS_LIBRARY}
  mysqlclient
  microhttpd
  ${GRPC_LINK_LIBS}
  ${PROTOBUF_LINK_LIB}
  yaml
  Threads::Threads
  m
  dl
)

if(TAOS_FOUND)
  list(APPEND COMMON_LIBRARIES ${TAOS_LIBRARY})
endif()

if(STREAM_FOUND)
  list(APPEND COMMON_LIBRARIES ${FFMPEG_LIBRARIES})
endif()

if(MOSQUITTO_FOUND)
  list(APPEND COMMON_LIBRARIES ${MOSQUITTO_LIBRARY})
endif()

target_link_libraries(main PRIVATE ${COMMON_LIBRARIES})

# Compiler options
target_compile_options(main PRIVATE
  $<$<COMPILE_LANGUAGE:C>:-Wall -Wextra -Wno-unused-parameter>
  $<$<COMPILE_LANGUAGE:CXX>:-Wall -Wextra -Wno-unused-parameter>
)

# Copy configuration file to build directory
configure_file(${CMAKE_CURRENT_SOURCE_DIR}/config.yaml
               ${CMAKE_CURRENT_BINARY_DIR}/config.yaml
               COPYONLY)

# Example programs
if(EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/examples)
  file(GLOB EXAMPLE_SRC examples/*.c)
  foreach(examplefile ${EXAMPLE_SRC})
    get_filename_component(examplename ${examplefile} NAME_WE)
    add_executable(example_${examplename} ${examplefile}
      ${COMMON_SOURCES}
    )
    target_link_libraries(example_${examplename} PRIVATE ${COMMON_LIBRARIES})
  endforeach()
endif()

# Installation rules
install(TARGETS main
  RUNTIME DESTINATION bin
)

install(DIRECTORY
  common/
  config/
  log/
  device/
  driver/
  DESTINATION include/mapper-framework-c
  FILES_MATCHING PATTERN "*.h"
)

# Print build configuration
message(STATUS "=== Build Configuration ===")
message(STATUS "C Standard: ${CMAKE_C_STANDARD}")
message(STATUS "CXX Standard: ${CMAKE_CXX_STANDARD}")
message(STATUS "Build Type: ${CMAKE_BUILD_TYPE}")
message(STATUS "Install Prefix: ${CMAKE_INSTALL_PREFIX}")
message(STATUS "")
message(STATUS "=== Feature Support Summary ===")
message(STATUS "Core Framework: YES")
message(STATUS "Device Management: YES")
message(STATUS "HTTP Server: YES")
message(STATUS "GRPC Server: YES")
message(STATUS "MySQL: YES")
message(STATUS "InfluxDB2: YES")
message(STATUS "Redis: YES")
if(TAOS_FOUND)
  message(STATUS "TDengine: YES")
else()
  message(STATUS "TDengine: NO (library not found)")
endif()
if(STREAM_FOUND)
  message(STATUS "Stream Processing: YES (FFmpeg)")
else()
  message(STATUS "Stream Processing: NO (FFmpeg not found)")
endif()
if(MOSQUITTO_FOUND)
  message(STATUS "MQTT Publish: YES")
else()
  message(STATUS "MQTT Publish: NO (mosquitto not found)")
endif()
message(STATUS "")
message(STATUS "=== Source Files Summary ===")
list(LENGTH COMMON_SOURCES SOURCE_COUNT)
message(STATUS "Total source files: ${SOURCE_COUNT}")
message(STATUS "================================")

# Debug information
if(CMAKE_BUILD_TYPE STREQUAL "Debug")
  message(STATUS "")
  message(STATUS "=== Debug Information ===")
  message(STATUS "Source files:")
  foreach(src ${COMMON_SOURCES})
    message(STATUS "  ${src}")
  endforeach()
  message(STATUS "Libraries:")
  foreach(lib ${COMMON_LIBRARIES})
    message(STATUS "  ${lib}")
  endforeach()
  message(STATUS "========================")
endif()