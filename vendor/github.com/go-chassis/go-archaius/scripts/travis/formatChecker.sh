diff -u <(echo -n) <(find . -name "*.go" -not -path ".git/*" -not -path "./third_party/*" | xargs gofmt -s -d)
if [ $? == 0 ]; then
	echo "Code is formatted properly"
	exit 0
else
	echo "Code is not formatted properly"
	exit 1
fi
