# Build a kubeedge release

The command 'make release' will build the binaries and zip them in a tar.gz file. Also
it will create a file with a checksum of each tar.gz file. At the moment the make file
will iterate through all given operation systems and the configured architectures. 

You can override the configured values for the operation systems and the architectures by
setting them as an environment variable. We use the variable OD\_RELEASE to define the
operation systems. The architectures can be defined on each binary release part. The parts
are keadm, edge and kubeedge. In the following table you can find the relation between
release part and the architecture variable. The variables can also be overwritten through 
environment variables.

| binary release part | variable name |
| ------------------- | ------------- |
| keadm | RELEASE\_KEADM\_ARCH |
| kubeedge | RELEASE\_KUBEEDGE\_ARCH |
| edge | RELEASE\_EDGE\_ARCH |

Also we can build each part separately. The following shows the relation between binary release part
and the make target.

| binary release part | make string |
| ------------------- | ----------- |
| keadm | create\_installer\_binaries |
| edge | create\_edge\_binaries |
| kubeedge | create\_kubeedge\_binaries |

To build the checksum the program shasum is defined with the SHA256 algorithm. This can easily
overwritten by setting the environment variable 'SHASUM'. Also you can change the default behavior
by change the value of the variable 'SHASUM' in the makefile.

## Contains

The different release part contains different 

keadm:
	- kubeedge binary (build from keadm)
	- version file

edge:
	- edge:
		- edge_core binary
		- conf directory
	- version file

kubeedge:
	- edge:
		- edge_core binary
		- conf directory
	- cloud:
		- edgecontroller
		- conf directory
	- version file

## Extend / Remove operations systems

### Extend operation systems

You have to add the name of the operation system, which is used by golang behind the current existing 
operation systems. An example is:
You want to build all files for a darwin system. (Darwin is used for OS-X(compare to OS-X-kernel))
```make
RELEASE_OS ?= "linux" "darwin"
```
Now all parts will be build for darwin and linux in all architectures.

### Remove an operation system

To remove an operation system you can easily remove the string from RELEASE\_OS.
In the next example, we will remove "darwin" from the list of operation systems.
```
RELEASE_OS ?= "linux"
```

## Extend / Remove Architectures

In this chapter I will explain how to extend or delete a build architecture. As described before in
the operations system part we use golang natives to build the operation system. You have to use the
golang specific names of the architectures. In the examples I will use the architecture 386.

### kubeedge
In this part I will explain how to add a build architecture for the kubeedge binaries.

#### Extend
You have to add the architecture on the variable RELEASE\_KUBEEDGE\_ARCH.
EXAMPLE:
```make
RELEASE_KUBEEDGE_ARCH ?= "arm" "amd64" "386"
```

#### Remove
You have to remove the architecture name from the variable RELEASE\_KUBEEDGE\_ARCH.
EXAMPLE:
```make
RELEASE_KUBEEDGE_ARCH ?= "arm" "amd64"
```

### edge
#### Extend
You have to add the architecture on the variable RELEASE\_EDGE\_ARCH.
EXAMPLE:
```make
RELEASE_EDGE_ARCH ?= "arm" "amd64" "386"
```

#### Remove
You have to remove the architecture name from the variable RELEASE\_EDGE\_ARCH.
EXAMPLE:
```make
RELEASE_EDGE_ARCH ?= "arm" "amd64"
```

### keadm
#### Extend
You have to add the architecture on the variable RELEASE\_KEADM\_ARCH.
EXAMPLE:
```make
RELEASE_KEADM_ARCH ?= "arm" "amd64" "386"
```

#### Remove
You have to remove the architecture name from the variable RELEASE\_KEADM\_ARCH.
EXAMPLE:
```
RELEASE_KEADM_ARCH ?= "arm" "amd64"
```
