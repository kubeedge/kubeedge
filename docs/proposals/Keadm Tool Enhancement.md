---
title: Keadm Tool Enhancement
authors:
- "@HT0403"
  approvers:
  creation-date: 2024-04-27
  last-updated: 2024-04-28
---

# Keadm Tool Enhancement

## Motivation

Keadm(KubeEdge installation tool) now only supports configuring a subset of parameters during EdgeCore installation.

We would like to support specifying parameters using `--set` or directly using an existing local configuration file to achieve full parameter configuration and meet the users' requirements. 

## Goals

- Use '--set' or local configuration file to set parameters

## Design Details

**The frame design drawing is as follows**：

![](../images/proposals/keadm%20tool%20enhancement.png)

1. **Start** - The entry point of the program which takes in two inputs：
   - `sets`:A string containing multiple key-value pairs separated by commas.
   - `Config`:A structure meant to hold the parsed configuration data.

2. **Split** - The `sets` string is split by commas into individual settings, and each setting is further split by the equal sign to separate keys from values.
3. **Parse Value** - Parses the type of each value, considering basic types such as integer (int), floating-point (float), string, and complex types like arrays.
4. **Decision** - Different processing is applied depending on the form of the key. There are three possible key types:
   - `key[...]` - The key indicates an index in an array.
   - `key[...].key[M]` - The key indicates a specific field within an element of the array.
   - `key[...].key[M].variable1` - The key indicates a variable within a specific field of an element in the array.
5. **Further Split** - The key is further split based on whether there is an index position or member variables.
6. **Use `reflect.ValueOf`** - Reflection is used to obtain the reflection value of the incoming structure pointer, then the `Elem` method is utilized to recursively find the reflection value of the field that needs modification.
7. **Key and Structure Type Check (`key && tStruct`)** - It determines if the key matches the fields in the structure.
8. **Modify Config** - The `Config` structure is modified according to the parsed content.
9. **End of Keys** - All keys have been processed.
10. **End** - The process ends.

