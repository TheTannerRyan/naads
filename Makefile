build:
	dep ensure
	go build -o client example/client.go

docker:
	dep ensure
	docker build -t naads .
	# docker save -o naads.tar naads # (save on one)
	# docker load -i naads.tar # (deploy on another)
	# docker run -d --restart=unless-stopped -p 80:6060 --name naads naads

clean:
	rm -rf vendor ./client

git: clean
	dep ensure

scan:
	snyk test
	snyk monitor

.SILENT:
