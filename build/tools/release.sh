#! /bin/sh
set -e

# this function will pint a help message
help () {
	echo "RELEASE script of kubeedge"
	echo
	echo "Usage:"
	echo -e "\tthe first argument should be the version, which should be released"
	echo -e "\tthe second argument is the path to the destination dir where the created files should be stored"
}

# this file will generate the checksum of an given file and store it into $filename.sum
# this scirpt will use shasum to generate the shasum over the file
# @param filename
generateChecksum () {
	file=$1
	filename=$2
	/usr/bin/shasum -a 512 $file  >> $filename
}

# this function will move the builded file and the the checksum to the destination directory
# @param destination directory
# @param filename
move () {
	desDir=$2
	filename=$1

	mkdir -p $desDir
	mv $filename $desDir/
}

# this function will generate the tar directories' sign the data and move the data to destDir
# @param version
# @param arch
# @param desDir
generateRel () {
	version=$1
	arch=$2
	desDir=$3

	keadm=$(echo keadm-$version-linux-$arch)
	edge=$(echo edge-$version-linux-$arch)
	kubeedge=$(echo kubeedge-$version-linux-$arch)
	edgesite=$(echo edgesite-$version-linux-$arch)

	cd $PWD 

	mkdir -p $edge/edge
	echo $version > $edge/version
	mkdir -p $kubeedge/edge
	echo $version > $kubeedge/version
	mkdir -p $kubeedge/cloud
	mkdir -p $kubeedge/keadm
	mkdir $keadm
	echo $version > $keadm/version
	mkdir -p $edgesite
	echo $version > $edgesite/version

	# copy edgecore to the expected directories
	echo copy edge
	cp edge/edgecore $edge/edge/
	cp -r edge/conf $edge/edge/
	cp edge/edgecore $kubeedge/edge/
	cp -r edge/conf $kubeedge/edge/

	# copy cloudcore to the expected directories
	cp cloud/cloudcore $kubeedge/cloud/
	cp -r cloud/conf $kubeedge/cloud/

	# copy keadm to the expected directories
	cp keadm/keadm $keadm/

	# copy to edgesite
	cp edgesite/edgesite $edgesite/
	cp -r edgesite/conf $edgesite/

	tar czfv $(echo $keadm.tar.gz) $keadm
	tar czfv $(echo $edge.tar.gz) $edge
	tar czfv $(echo $kubeedge.tar.gz) $kubeedge
	tar czfv $(echo $edgesite.tar.gz) $edgesite
	ls *.gz

	echo generate checksum
	filename=$(echo checksum_$arch.txt)
	generateChecksum $(echo $edge.tar.gz) $filename
	generateChecksum $(echo $keadm.tar.gz) $filename
	generateChecksum $(echo $kubeedge.tar.gz) $filename
	generateChecksum $(echo $edgesite.tar.gz) $filename

	move $(echo $keadm.tar.gz) $desDir
	move $(echo $edge.tar.gz) $desDir
	move $(echo $kubeedge.tar.gz) $desDir
	move $(echo $edgesite.tar.gz) $desDir
	filename=$(echo checksum_$arch.txt)
	move $filename $desDir

	rm -rf $edge
	rm -rf $keadm
	rm -rf $kubeedge
	rm -rf $edgesite
	rm -rf $(echo $edge.tar.gz)
	rm -rf $(echo $keadm.tar.gz)
	rm -rf $(echo $kubeedge.tar.gz)
	rm -rf $(echo $edgesite.tar.gz)
	rm -rf $filename
}

if [ $# -ne 2 ]; then
	help
	exit 2
fi

version=$1
desDir=$2

# build x86_64
make
generateRel $version x86_64 $desDir


# build arch arm v6
make cross_build
generateRel $version armv6 $desDir

# build arch armv7
make cross_build_v7
generateRel $version armv7 $desDir
