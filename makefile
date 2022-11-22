compile:
	echo "Compiling for Raspberry PI"
	GOOS=linux GOARCH=arm go build -o bin/main-raspberry main.go