build:
	dep ensure
	go build -o client example/client.go

clean:
	rm -rf vendor ./client

git: clean
	dep ensure

scan:
	snyk test
	snyk monitor

.SILENT:
