.PHONY: build cover start test test-integration deploy build-docker

export image := `aws lightsail get-container-images --service-name canvas | jq -r '.containerImages[0].image'`

# build docker command
build:
	@echo "Building..."
	@templ generate
	@go build -o main cmd/server/main.go

cover:
	go tool cover -html=cover.out

start:
	go run cmd/server/*.go

test:
	go test -coverprofile=cover.out -short ./...

test-integration:
	go test -coverprofile=cover.out -p 1 ./...

watch:
	air

deploy:
	docker build -t canvas .
	aws lightsail push-container-image --service-name canvas --label app --image canvas
	aws lightsail create-container-service-deployment --service-name canvas \
		--containers '{"app":{"image":"'$(image)'","environment":{"HOST":"","PORT":"8080","LOG_ENV":"production"},"ports":{"8080":"HTTP"}}}' \
		--public-endpoint '{"containerName":"app","containerPort":8080,"healthCheck":{"path":"/health"}}'
