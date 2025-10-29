#ifndef COMMON_DATAMETHOD_H
#define COMMON_DATAMETHOD_H

#include <stddef.h>

// Parameter defines a parameter for a method.
typedef struct {
    char *propertyName; 
    char *valueType;  
} Parameter;

// Method defines a device method.
typedef struct {
    char *name;          
    char *path; 
    Parameter *parameters; 
    size_t parametersCount; 
} Method;

// DataMethod defines standard model for deviceMethod.
typedef struct {
    Method *methods;
    size_t methodsCount;
} DataMethod;

#endif // DATAMETHOD_H