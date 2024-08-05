FROM debian:latest
RUN apt update && apt install -y git golang
RUN apt install -y make
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe
COPY . .
RUN rm go.mod
RUN rm go.sum
EXPOSE 8091
CMD ["./start.sh"]
