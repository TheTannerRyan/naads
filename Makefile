build:
	dep ensure
	go build -o client example/client.go

docker:
	dep ensure
	docker build -t naads .
	docker run -d --restart=unless-stopped -p 6060:6060 --name naads naads

clean:
	rm -rf vendor ./client

git: clean
	dep ensure

scan:
	snyk test
	snyk monitor

.SILENT:
