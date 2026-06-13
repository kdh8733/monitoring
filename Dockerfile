# --- build ---
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
# stdlib-only module: no external deps to download.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- runtime (distroless, nonroot) ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
