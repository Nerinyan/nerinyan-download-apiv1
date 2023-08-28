FROM scratch
ARG TARGETARCH
ARG TARGETOS
COPY ./ca-certificates.crt /etc/ssl/certs/
COPY ./app-${TARGETOS}-${TARGETARCH} /app
ENTRYPOINT ["/app"]
