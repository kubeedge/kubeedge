# gomonkey

gomonkey is a library to make monkey patching in unit tests easy.

## Features

+ support a patch for a function
+ support a patch for a member method
+ support a patch for a interface
+ support a patch for a function variable
+ support a patch for a global variable
+ support patches of a specified sequence for a function
+ support patches of a specified sequence for a member method
+ support patches of a specified sequence for a interface
+ support patches of a specified sequence for a function variable

## Notes
+ gomonkey fails to patch a function or a member method if inlining is enabled, please running your tests with inlining disabled by adding the command line argument that is `-gcflags=-l`(below go1.10) or `-gcflags=all=-l`(go1.10 and above).
+ gomonkey should work on any amd64 system.
+ A panic may happen when a goroutine is patching a function or a member method that is visited by another goroutine at the same time. That is to say, gomonkey is not threadsafe.
+ go1.6 version of the reflection mechanism supports the query of private member methods, but go1.7 and above does not support it. However, all versions of the reflection mechanism support the query of private functions, so gomonkey will trigger a `panic` for only patching a private member method when go1.7 and above is used.


## Supported Platform:

- MAC OS X amd64
- Linux amd64
- Windows amd64

## Installation
```go
$ go get github.com/agiledragon/gomonkey
```
## Using gomonkey

Please refer to the test cases as idioms, very complete and detailed.

