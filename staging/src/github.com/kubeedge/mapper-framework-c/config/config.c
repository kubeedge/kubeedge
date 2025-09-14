#include "config/config.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <yaml.h>
#include <strings.h>  // Added: strcasecmp

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

    // Initialize default values inside config_parse
    memset(&cfg->database, 0, sizeof(cfg->database));
    // cfg->database.mysql.enabled = 0 (disabled by default)

    yaml_parser_t parser;
    yaml_token_t token;
    char key[128] = {0};
    int in_grpc_server = 0, in_common = 0;
    int in_database = 0, in_mysql = 0;  // Added

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
                strncpy(key, (char *)token.data.scalar.value, sizeof(key) - 1);
                key[sizeof(key) - 1] = '\0';
            }
            else {
                yaml_token_delete(&token);
                continue;
            }

            //skip VALUE token (:) before checking whether it's a sub-mapping or a scalar value
            yaml_token_delete(&token);
            do {
                yaml_parser_scan(&parser, &token);
            } while (token.type == YAML_VALUE_TOKEN);

            if (token.type == YAML_BLOCK_MAPPING_START_TOKEN) {
                // Enter sub-mapping
                if (strcmp(key, "grpc_server") == 0) {
                    in_grpc_server = 1; in_common = 0; in_database = 0; in_mysql = 0;
                } else if (strcmp(key, "common") == 0) {
                    in_common = 1; in_grpc_server = 0; in_database = 0; in_mysql = 0;
                } else if (strcmp(key, "database") == 0) {
                    in_database = 1; in_common = 0; in_grpc_server = 0; in_mysql = 0;
                } else if (in_database && strcmp(key, "mysql") == 0) {
                    in_mysql = 1;
                }
                yaml_token_delete(&token);
                continue;
            }

            // If it is a scalar value, write it according to the current context
            if (token.type == YAML_SCALAR_TOKEN)
            {
                if (in_grpc_server) {
                    if (strcmp(key, "socket_path") == 0)
                        strncpy(cfg->grpc_server.socket_path, (char *)token.data.scalar.value, sizeof(cfg->grpc_server.socket_path) - 1);
                }
                else if (in_common) {
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
                else if (in_mysql) {
                    if (strcmp(key, "enabled") == 0) {
                        const char *v = (char *)token.data.scalar.value;
                        cfg->database.mysql.enabled = (!strcasecmp(v,"true") || !strcmp(v,"1")) ? 1 : 0;
                    } else if (strcmp(key, "addr") == 0) {
                        strlcpy(cfg->database.mysql.addr, (char *)token.data.scalar.value, sizeof(cfg->database.mysql.addr));
                    } else if (strcmp(key, "database") == 0) {
                        strlcpy(cfg->database.mysql.database, (char *)token.data.scalar.value, sizeof(cfg->database.mysql.database));
                    } else if (strcmp(key, "username") == 0) {
                        strlcpy(cfg->database.mysql.username, (char *)token.data.scalar.value, sizeof(cfg->database.mysql.username));
                    } else if (strcmp(key, "password") == 0) {
                        strlcpy(cfg->database.mysql.password, (char *)token.data.scalar.value, sizeof(cfg->database.mysql.password));
                    } else if (strcmp(key, "port") == 0) {
                        cfg->database.mysql.port = atoi((char *)token.data.scalar.value);
                    } else if (strcmp(key, "ssl_mode") == 0) {  // Added parsing
                        strlcpy(cfg->database.mysql.ssl_mode, (char *)token.data.scalar.value, sizeof(cfg->database.mysql.ssl_mode));
                    }
                }
                yaml_token_delete(&token);
            } else {
                yaml_token_delete(&token);
            }
        }
        else if (token.type == YAML_BLOCK_END_TOKEN) {
            // Exit sub-mapping
            if (in_mysql) { in_mysql = 0; }
            else if (in_database) { in_database = 0; }
            else if (in_common) { in_common = 0; }
            else if (in_grpc_server) { in_grpc_server = 0; }
            yaml_token_delete(&token);
        }
        else {
            yaml_token_delete(&token);
        }
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