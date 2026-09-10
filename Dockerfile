# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/image-trust-demo .

FROM scratch
COPY --from=build /out/image-trust-demo /image-trust-demo
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/image-trust-demo"]
