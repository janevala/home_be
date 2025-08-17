FROM golang:1.24
RUN apt update
RUN apt install -y make
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe
COPY . .
RUN rm -f go.mod
RUN rm -f go.sum
EXPOSE 8091
CMD ["./start.sh"]
