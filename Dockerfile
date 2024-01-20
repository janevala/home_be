FROM debian:latest
RUN apt update && apt install -y git golang
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe
COPY . .
EXPOSE 8091
CMD ["go", "run", "main.go"]
