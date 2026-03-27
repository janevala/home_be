# News Backend
 
[Documentation](https://github.com/janevala/home_fe/blob/main/DOC.md)

## Get Started
- Makefile
- .vscode/launch.json
- Dockerfile & build.sh

```bash
docker build --no-cache -f Dockerfile -t news-backend .
docker run --name api-host --network home-network -p 7071:7071 --restart always -d news-backend
```