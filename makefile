build:
	go build -o ./.bin/bot cmd/main.go

run: build
	./.bin/bot

build-image:
	docker build -t telegram-bot-youtube:v0.1 .

start-container:
	docker run --name telegram-bot-youtube --env-file .env telegram-bot-youtube:v0.1