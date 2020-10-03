# Cross Compiling KubeEdge

In most of the cases, when you are trying to compile KubeEdge edgecore on Raspberry Pi or any other device, you may run out of memory, in that case, it is advisable to cross-compile the Edgecore binary and transfer it to your edge device.

## For ARM Architecture from x86 Architecture

Clone KubeEdge

```shell
# Build and run KubeEdge on an ARMv6 target device.

git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
sudo apt-get install gcc-arm-linux-gnueabi
export GOARCH=arm
export GOOS="linux"
export GOARM=6 #Pls give the appropriate arm version of your device
export CGO_ENABLED=1
export CC=arm-linux-gnueabi-gcc
make edgecore
```

If you are compiling KubeEdge edgecore for Raspberry Pi and check the [Makefile](/Makefile) for the edge.

In that CC has been defined as
```
export CC=arm-linux-gnueabi-gcc;
```

However, it always good to check what's your gcc on Raspberry Pi says by

```
gcc -v

Using built-in specs.
COLLECT_GCC=gcc
COLLECT_LTO_WRAPPER=/usr/lib/gcc/arm-linux-gnueabihf/6/lto-wrapper
Target: arm-linux-gnueabihf
Configured with: ../src/configure -v --with-pkgversion='Raspbian 6.3.0-18+rpi1+deb9u1' --with-bugurl=file:///usr/share/doc/gcc-6/README.Bugs --enable-languages=c,ada,c++,java,go,d,fortran,objc,obj-c++ --prefix=/usr --program-suffix=-6 --program-prefix=arm-linux-gnueabihf- --enable-shared --enable-linker-build-id --libexecdir=/usr/lib --without-included-gettext --enable-threads=posix --libdir=/usr/lib --enable-nls --with-sysroot=/ --enable-clocale=gnu --enable-libstdcxx-debug --enable-libstdcxx-time=yes --with-default-libstdcxx-abi=new --enable-gnu-unique-object --disable-libitm --disable-libquadmath --enable-plugin --with-system-zlib --disable-browser-plugin --enable-java-awt=gtk --enable-gtk-cairo --with-java-home=/usr/lib/jvm/java-1.5.0-gcj-6-armhf/jre --enable-java-home --with-jvm-root-dir=/usr/lib/jvm/java-1.5.0-gcj-6-armhf --with-jvm-jar-dir=/usr/lib/jvm-exports/java-1.5.0-gcj-6-armhf --with-arch-directory=arm --with-ecj-jar=/usr/share/java/eclipse-ecj.jar --with-target-system-zlib --enable-objc-gc=auto --enable-multiarch --disable-sjlj-exceptions --with-arch=armv6 --with-fpu=vfp --with-float=hard --enable-checking=release --build=arm-linux-gnueabihf --host=arm-linux-gnueabihf --target=arm-linux-gnueabihf
Thread model: posix
gcc version 6.3.0 20170516 (Raspbian 6.3.0-18+rpi1+deb9u1)
```

If you see, Target has been defined as
```
Target: arm-linux-gnueabihf
```
in that case, export CC as
```
arm-linux-gnueabihf-gcc rather than arm-linux-gnueabi-gcc
```

Also, based on the above result, you may have to install
```
gcc-arm-linux-gnueabi - GNU C cross-compiler for architecture armel

or

gcc-arm-linux-gnueabihf - GNU C cross-compiler for architecture armhf
```
