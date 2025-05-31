# Home BE

Home backend application, to be used together with Home frontend (Flutter client app).

Home BE is app written in Golang. Its intended to provide authentication for client, and then after login, RSS news resources.

It is simple demo app for learning purposes.

Go propgram runs as a microservice in Docker container, and listens port 8091.

Configure sites.json, and add/remove feed providers. Configure database.json, for storage connection.

Notes bellow give reference for setting up the containers.

Separate Home BE Crawler is running as a different microservice, and crawls RSS feeds online. This Programs makes them available through various APIs.

AI integration to Ollama local AI instance is integrated through Talkative plugin for Golang.

# Go notes
```
sudo apt install -y golang
go mod init github.com/janevala/home_be
go mod tidy
go get github.com/mmcdole/gofeed
go get github.com/google/uuid
go get github.com/gorilla/mux
go get github.com/lib/pq
go get github.com/rifaideen/talkative
go get github.com/graphql-go/graphql
```

# Docker notes
```
sudo docker network create home-network

sudo docker build --no-cache -f Dockerfile -t news-backend .
sudo docker run --name api-host --network home-network -p 8091:8091 -d news-backend

sudo docker network connect home-network api-host
```
