diff -u <(echo -n) <(find . -name "*.go" -not -path ".git/*" | grep -v _test | xargs gocyclo -over 16)
if [ $? == 0 ]; then
	echo "All function has less cyclomatic complexity..."
	exit 0
else
	echo "Fucntions/function has more cyclomatic complexity..."
	exit 1
fi
