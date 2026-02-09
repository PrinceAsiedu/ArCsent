FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/arcsent ./cmd/arcsent

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/arcsent /app/arcsent
COPY configs/config.json /etc/arcsent/config.json
USER nonroot:nonroot
EXPOSE 8787 8788
ENTRYPOINT ["/app/arcsent", "-config", "/etc/arcsent/config.json"]
