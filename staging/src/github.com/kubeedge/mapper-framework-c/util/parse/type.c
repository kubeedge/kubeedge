#include "util/parse/type.h"
#include <stdlib.h>
#include <string.h>

static char *strdup_safe(const char *s) {
    if (!s) return NULL;
    char *copy = malloc(strlen(s) + 1);
    if (copy) strcpy(copy, s);
    return copy;
}

// Twin数组 -> gRPC Twin数组
V1beta1__Twin **ConvTwinsToGrpc(const Twin *twins, int twin_count, int *out_count) {
    V1beta1__Twin **res = malloc(sizeof(V1beta1__Twin*) * twin_count);
    for (int i = 0; i < twin_count; ++i) {
        res[i] = malloc(sizeof(V1beta1__Twin));
        v1beta1__twin__init(res[i]);
        res[i]->propertyname = strdup_safe(twins[i].propertyName);

        // ObservedDesired
        res[i]->observeddesired = malloc(sizeof(V1beta1__TwinProperty));
        v1beta1__twin_property__init(res[i]->observeddesired);
        res[i]->observeddesired->value = strdup_safe(twins[i].observedDesired.value);
        res[i]->observeddesired->n_metadata = 2;
        res[i]->observeddesired->metadata = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry*) * 2);
        for (int k = 0; k < 2; ++k) {
            res[i]->observeddesired->metadata[k] = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry));
            v1beta1__twin_property__metadata_entry__init(res[i]->observeddesired->metadata[k]);
        }
        res[i]->observeddesired->metadata[0]->key = strdup_safe("type");
        res[i]->observeddesired->metadata[0]->value = strdup_safe(twins[i].observedDesired.metadata.type);
        res[i]->observeddesired->metadata[1]->key = strdup_safe("timestamp");
        res[i]->observeddesired->metadata[1]->value = strdup_safe(twins[i].observedDesired.metadata.timestamp);

        // Reported
        res[i]->reported = malloc(sizeof(V1beta1__TwinProperty));
        v1beta1__twin_property__init(res[i]->reported);
        res[i]->reported->value = strdup_safe(twins[i].reported.value);
        res[i]->reported->n_metadata = 2;
        res[i]->reported->metadata = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry*) * 2);
        for (int k = 0; k < 2; ++k) {
            res[i]->reported->metadata[k] = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry));
            v1beta1__twin_property__metadata_entry__init(res[i]->reported->metadata[k]);
        }
        res[i]->reported->metadata[0]->key = strdup_safe("type");
        res[i]->reported->metadata[0]->value = strdup_safe(twins[i].reported.metadata.type);
        res[i]->reported->metadata[1]->key = strdup_safe("timestamp");
        res[i]->reported->metadata[1]->value = strdup_safe(twins[i].reported.metadata.timestamp);
    }
    if (out_count) *out_count = twin_count;
    return res;
}

// gRPC Twin数组 -> Twin数组
Twin *ConvGrpcToTwins(V1beta1__Twin **twins, int twin_count, const Twin *src_twins, int src_count, int *out_count) {
    Twin *res = malloc(sizeof(Twin) * twin_count);
    int res_count = 0;
    for (int i = 0; i < twin_count; ++i) {
        const char *name = twins[i]->propertyname;
        int found = 0;
        for (int j = 0; j < src_count; ++j) {
            if (strcmp(name, src_twins[j].propertyName) == 0) {
                res[res_count] = src_twins[j]; // 浅拷贝
                found = 1;
                break;
            }
        }
        if (!found) continue;
        // ObservedDesired
        if (twins[i]->observeddesired) {
            res[res_count].observedDesired.value = strdup_safe(twins[i]->observeddesired->value);
            for (size_t k = 0; k < twins[i]->observeddesired->n_metadata; ++k) {
                if (strcmp(twins[i]->observeddesired->metadata[k]->key, "type") == 0)
                    res[res_count].observedDesired.metadata.type = strdup_safe(twins[i]->observeddesired->metadata[k]->value);
                if (strcmp(twins[i]->observeddesired->metadata[k]->key, "timestamp") == 0)
                    res[res_count].observedDesired.metadata.timestamp = strdup_safe(twins[i]->observeddesired->metadata[k]->value);
            }
        }
        // Reported
        if (twins[i]->reported) {
            res[res_count].reported.value = strdup_safe(twins[i]->reported->value);
            for (size_t k = 0; k < twins[i]->reported->n_metadata; ++k) {
                if (strcmp(twins[i]->reported->metadata[k]->key, "type") == 0)
                    res[res_count].reported.metadata.type = strdup_safe(twins[i]->reported->metadata[k]->value);
                if (strcmp(twins[i]->reported->metadata[k]->key, "timestamp") == 0)
                    res[res_count].reported.metadata.timestamp = strdup_safe(twins[i]->reported->metadata[k]->value);
            }
        }
        ++res_count;
    }
    if (out_count) *out_count = res_count;
    return res;
}

// MsgTwin map -> gRPC Twin数组
V1beta1__Twin **ConvMsgTwinToGrpc(const char **names, MsgTwin **msgTwins, int count, int *out_count) {
    V1beta1__Twin **res = malloc(sizeof(V1beta1__Twin*) * count);
    for (int i = 0; i < count; ++i) {
        res[i] = malloc(sizeof(V1beta1__Twin));
        v1beta1__twin__init(res[i]);
        res[i]->propertyname = strdup_safe(names[i]);
        // Reported
        res[i]->reported = malloc(sizeof(V1beta1__TwinProperty));
        v1beta1__twin_property__init(res[i]->reported);
        res[i]->reported->value = strdup_safe(msgTwins[i]->actual->value);
        res[i]->reported->n_metadata = 2;
        res[i]->reported->metadata = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry*) * 2);
        for (int k = 0; k < 2; ++k) {
            res[i]->reported->metadata[k] = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry));
            v1beta1__twin_property__metadata_entry__init(res[i]->reported->metadata[k]);
        }
        res[i]->reported->metadata[0]->key = strdup_safe("type");
        res[i]->reported->metadata[0]->value = strdup_safe(msgTwins[i]->metadata->type);
        res[i]->reported->metadata[1]->key = strdup_safe("timestamp");
        res[i]->reported->metadata[1]->value = strdup_safe(msgTwins[i]->actual->metadata.timestamp);

        // ObservedDesired
        res[i]->observeddesired = malloc(sizeof(V1beta1__TwinProperty));
        v1beta1__twin_property__init(res[i]->observeddesired);
        res[i]->observeddesired->value = strdup_safe(msgTwins[i]->expected->value);
        res[i]->observeddesired->n_metadata = 2;
        res[i]->observeddesired->metadata = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry*) * 2);
        for (int k = 0; k < 2; ++k) {
            res[i]->observeddesired->metadata[k] = malloc(sizeof(V1beta1__TwinProperty__MetadataEntry));
            v1beta1__twin_property__metadata_entry__init(res[i]->observeddesired->metadata[k]);
        }
        res[i]->observeddesired->metadata[0]->key = strdup_safe("type");
        res[i]->observeddesired->metadata[0]->value = strdup_safe(msgTwins[i]->metadata->type);
        res[i]->observeddesired->metadata[1]->key = strdup_safe("timestamp");
        res[i]->observeddesired->metadata[1]->value = strdup_safe(msgTwins[i]->actual->metadata.timestamp);
    }
    if (out_count) *out_count = count;
    return res;
}