diff -u <(echo -n) <(find . -type d -not -path "./third_party/*" |grep -v examples | xargs deadcode)
if [ $? == 0 ]; then
	echo "No Deadcode"
	exit 0
else
	echo "Deadcode found ... Remove the unused code"
	exit 1
fi
