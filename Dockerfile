FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod main.go ./
COPY web ./web
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/master-stb .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends adb ca-certificates && rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 --create-home app
COPY --from=build /out/master-stb /usr/local/bin/master-stb
COPY web/index.html /opt/master-stb/index.html
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/master-stb"]
