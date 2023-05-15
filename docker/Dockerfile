FROM golang:1.18 AS build

RUN --mount=type=secret,id=GITHUB_TOKEN,required \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN):@github.com/".insteadOf "https://github.com/"

WORKDIR /src

COPY go.mod go.sum /src/

COPY . /src/

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/server cmd/server/main.go \
 && CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/migration cmd/migration/main.go \
 && CGO_ENABLED=0 GOOS=linux go build -a -tags musl -installsuffix cgo -o /app/server-grpc cmd/grpc/main.go

FROM gcr.io/distroless/static-debian11 AS release

WORKDIR /app

COPY --from=build /app/ /app/
COPY --from=build /src/schema/migration/ /app/schema/migration/
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.17 /ko-app/grpc-health-probe /bin/grpc-health-probe

CMD [ "./server" ]
