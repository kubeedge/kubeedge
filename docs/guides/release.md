# Release binaries

## Table of Contents
* [Introduction](#introduction)
* [Generate](#generate)
* [Add architecture](#add architecture)
* [Remove architecture](#remove architecture)

## Introduction
This section describes, how we can build the release binaries. 
In the current state we will build the release for the following architectures:
- armV6
- armV7
- x86\_64

We are using a seperate release script in the build/tool directory.

## Generate
To generate a current release you can build all the files from hand or using this script. You can execute from the root directory of this repository via this command:
```bash
bash build/tools/release.sh $version $destinationDir
```

In the destination directory you fill find the following files:
- keadm-$version-linux-x86\_64
- keadm-$version-linux-armV6
- keadm-$version-linux-armV7
- edge-$version-linux-x86\_64
- edge-$version-linux-armV6
- edge-$version-linux-armV7
- kubeedge-$version-linux-x86\_64
- kubeedge-$version-linux-armV6
- kubeedge-$version-linux-armV7
- edgesite-$version-linux-x86\_64
- edgesite-$version-linux-armV6
- edgesite-$version-linux-armV7
- the checksum files to every builded binary package


## Add architecture
To add an architecture you have to add the make target at the bottom of the release.sh file and after this executing generateRel with the parameter version the builded architecture and the destination dir.

## Remove architecture
To remove an architecture you have to remove the make target and the following function call of generateRel
