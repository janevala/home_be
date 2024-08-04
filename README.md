# Home BE

Home backend application, to be used together with Home frontend (Flutter client app).

Home BE is app written in Golang. Its intended to provide authentication for client, and then after login, RSS news resources.

It is simple demo app for learning purposes.

This app is meant to be used in Docker container, and runs by default on port 8091.

Configure sites.json, and add/remove feed providers. Configure database.json, for storage connection.

Notes bellow give reference for setting up the containers.

# Go notes
```
sudo apt install -y golang
go mod init github.com/janevala/home_be
go mod tidy
go get github.com/mmcdole/gofeed
go get github.com/google/uuid
go get github.com/gorilla/mux
go get github.com/lib/pq

```

# Docker notes
```
sudo docker run --name postgres-container -e POSTGRES_PASSWORD=1234 -p 5432:5432 -d postgres
sudo docker exec -ti postgres-container createdb -U postgres homebedb
sudo docker exec -ti postgres-container psql -U postgres
postgres=# \c homebedb
homebedb=# \q

homebedb=# \dt feed_items

sudo docker run -d -p 8091:8091 home-backend
sudo docker build --no-cache -f Dockerfile -t home-backend .
```
