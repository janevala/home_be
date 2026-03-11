# News Backend

# Development setup (modules are listed in Makefile)
```
sudo apt install -y golang make
go mod init github.com/janevala/home_be
make build
```

# Environments & Go build tags & make targets
- release
- debug

# Startup scenarios, need to handle debug vs release tag
- Makefile
- Dockerfile & build.sh
- VSCode launch.json

# API Endpoints

## Notes
- All endpoints require the `code=123` query parameter
- CORS is enabled for all endpoints
- The server runs on port 7071 by default


# Docker notes
```
sudo docker network create home-network
sudo docker build --no-cache -f Dockerfile -t news-backend .
sudo docker run --name api-host --network home-network -p 7071:7071 --restart always -d news-backend
sudo docker network connect home-network api-host
```

# Https on VPS
```
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
sudo vi /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

- Check Caddyfile for configuration