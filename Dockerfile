# Multi-stage Hermesd + Mission Control
FROM node:20-alpine AS ui
WORKDIR /ui
COPY mission-control/package*.json ./
RUN npm ci
COPY mission-control/ ./
RUN npm run build

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY kernel/go.mod kernel/go.sum* ./
RUN go mod download
COPY kernel/ ./
RUN CGO_ENABLED=0 go build -o /hermesd ./cmd/hermesd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates bash
WORKDIR /app
COPY --from=build /hermesd /app/hermesd
COPY --from=ui /ui/dist /app/mission-control/dist
COPY plugins /app/plugins
ENV HERMES_UI_DIST=/app/mission-control/dist
ENV HERMES_PLUGINS=/app/plugins
EXPOSE 8080
ENTRYPOINT ["/app/hermesd", "serve", ":8080"]
