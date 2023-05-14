FROM golang:1.18 AS base

ARG GITHUB_TOKEN

RUN apt-get update && apt-get install -y --no-install-recommends mariadb-client \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/*

RUN git config --global url."https://$GITHUB_TOKEN:@github.com/".insteadOf "https://github.com/"

WORKDIR /src

COPY go.mod go.sum /src/

RUN GOPRIVATE=github.com/hungdv136/rio/internal go mod download

COPY . /src/

CMD ["./docker/test.sh"]
