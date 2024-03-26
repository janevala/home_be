# Home BE
Home backend application, to be used together with Home frontend (Flutter client app).

Home BE is app written in Golang. Its inded to provide authentication for client, as well as news resources. This app is intended to be used in Docker container, and runs by default on port 8091. It is simple demo app for learning purposes.

Configure sites.json, and add/remove feed providers.

# Go notes
```
sudo apt install -y golang
go mod init github.com/janevala/home_be
go mod tidy
go get github.com/mmcdole/gofeed
go get github.com/google/uuid
go get github.com/gorilla/mux
```

# Docker notes
```
sudo docker run -d -p 8091:8091 home-backend
sudo docker build --no-cache -f Dockerfile -t home-backend .
```
