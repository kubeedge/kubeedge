#ifndef COMMON_DATAMETHOD_H
#define COMMON_DATAMETHOD_H

#include <stddef.h>

// Parameter defines a parameter for a method.
typedef struct {
    char *propertyName; // Name of the property
    char *valueType;    // Value type of the property
} Parameter;

// Method defines a device method.
typedef struct {
    char *name;           // Method name
    char *path;           // Method path
    Parameter *parameters; // Array of parameters
    size_t parametersCount; // Number of parameters
} Method;

// DataMethod defines standard model for deviceMethod.
typedef struct {
    Method *methods;      // Array of methods
    size_t methodsCount;  // Number of methods
} DataMethod;

#endif // DATAMETHOD_H