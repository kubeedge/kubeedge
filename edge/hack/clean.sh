#!/bin/bash

# **clean.sh**

cd $GOPATH/src/github.com/kubeedge/kubeedge/edge

#removes bin folder which is created after running make verify
remove_folder() {
if [ -d bin ]; then
rm -rf bin
fi
}

#terminate edge core if running
stop_edgecore () {
if pgrep edgecore >/dev/null 2>&1 ; then
     pkill edgecore
fi
}

#delete logs,covergage and database related files
cleanup_files () {
if [ -f edgecore ]; then
rm -f edgecore
fi

find . -type f -name "*db" -exec rm -f {} \;
find . -type f -name "*log" -exec rm -f {} \;
find . -type f -name "*out" -exec rm -f {} \;
}

remove_folder
stop_edgecore
cleanup_files
