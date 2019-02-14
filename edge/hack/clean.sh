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
stop_edge_core () {
if pgrep edge_core >/dev/null 2>&1 ; then
     pkill -9 edge_core
fi
}

#delete logs,covergage and database related files
cleanup_files () {
if [ -f edge_core ]; then
rm -f edge_core
fi

find . -type f -name "*db" -exec rm -f {} \;
find . -type f -name "*log" -exec rm -f {} \;
find . -type f -name "*out" -exec rm -f {} \;
}

remove_folder
stop_edge_core
cleanup_files
