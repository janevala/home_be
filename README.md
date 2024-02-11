# homebe
Home BE Auth, to be used together with Home FE which is a Flutter cient app. Home BE is backend app written in Go. This backend app can be run in Docker container to provide client app with data. It runs by default on port 8091, and is not using true authentication, as it is not intended to be used in production. It is intended to be used as a demo app for learning purposes.


## Go
```
sudo apt install -y golang
go mod init github.com/janevala/home_be
go mod tidy
go get github.com/mmcdole/gofeed
go get github.com/google/uuid
```

# Docker
```
sudo docker run -d -p 8091:8091 home-backend
sudo docker build --no-cache -f Dockerfile -t home-backend .
```
