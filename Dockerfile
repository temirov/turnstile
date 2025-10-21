FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/ets .

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/ets /ets
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/ets"]
