# Home BE

Home backend application, to be used together with Home frontend (Flutter client app).

Home BE is app written in Golang. Its intended to provide authentication for client, and then after login, RSS news resources.

It is simple demo app for learning purposes.

Go propgram runs as a microservice in Docker container, and listens port 7071.

Configure sites.json, and add/remove feed providers. Configure database.json, for storage connection.

Notes bellow give reference for setting up the containers.

Separate Home BE Crawler is running as a different microservice, and crawls RSS feeds online. This Programs makes them available through various APIs.

AI integration to Ollama local AI instance is integrated through Talkative plugin for Golang.

# Environments & Go build tags & make targets
- release
- debug

# Startup scenarios, need to handle debug vs release tag
- Makefile
- Dockerfile & start.sh
- VSCode launch.json

# API Endpoints

The application provides the following REST API endpoints:

## Authentication
- `POST /auth` - Authenticate user
  - Request body: `{"username": "123", "password": "123"}`
  - Returns: 200 OK on success, 401 Unauthorized on failure

## RSS Feeds
- `GET /sites?code=123` - Get list of configured RSS feed sources
  - Returns: JSON array of feed sources with UUID, title, and URL
  - Example response:
    ```json
    {
      "time": 1711273868253,
      "title": "RSS Feeds",
      "sites": [
        {
          "title": "Phoronix",
          "url": "https://www.phoronix.com/rss.php"
        }
      ]
    }
    ```

- `GET /archive?code=123` - Get archived/processed RSS feed items
  - Returns: JSON array of feed items with title, description, link, publication date, etc.
  - Items are sorted by publication date (newest first)

## AI Integration
- `POST /explain?code=123` - Query the AI for explanations
  - Request body: `{"query": "your question here"}`
  - Returns: AI-generated response
  - Note: Requires local Ollama server running on port 11434
  - LLM: translategemma:12b https://ollama.com/library/translategemma/tags

## Notes
- All endpoints require the `code=123` query parameter for authentication
- CORS is enabled for all endpoints
- The server runs on port 7071 by default

# Go notes (modules are listed in Makefile)
```
sudo apt install -y golang
go mod init github.com/janevala/home_be
make build
```

# Docker notes
```
sudo docker network create home-network

sudo docker build --no-cache -f Dockerfile -t news-backend .
sudo docker run --name api-host --network home-network -p 7071:7071 --restart always -d news-backend

sudo docker network connect home-network api-host
```
