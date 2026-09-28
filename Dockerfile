# Cross-compile on the build host rather than emulating the target platform
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY public ./public
# Runs natively on the build host; the live integration tests run on a schedule instead
ENV CGO_ENABLED=0
RUN go test ./...
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /superquiz .

FROM scratch
# Needed to verify TLS for the sites the proxy fetches
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /superquiz /superquiz
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/superquiz"]
