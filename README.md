# homebe
Home BE Auth, to be used together with Home FE which is a Flutter cient app. Home BE is backend app written in Go. This backend app can be run in Docker container to provide client app with data. It runs by default on port 8091, and is not using true authentication, as it is not intended to be used in production. It is intended to be used as a demo app for learning purposes.

# Before running Docker, make sure local build works
 4138  sudo docker run -d -p 8091:8091 homebe
 4139  sudo docker build --no-cache -f Dockerfile -t homebe .


## Installation notes

export GOROOT=/Users/user/go
export GOPATH=$GOROOT
export PATH=$PATH:/Users/user/flutter/bin:/Users/user/flutter/bin/cache/dart-sdk/bin:$GOROOT/bin

191883  go mod init main.go
191884  go mod tidy
191887  go get github.com/mmcdole/gofeed
191889  go get github.com/google/uuid
