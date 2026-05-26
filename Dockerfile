FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/expenses-tracker .

FROM alpine:3.22

LABEL org.opencontainers.image.source="https://github.com/kevinssheva/expenses-tracker"
LABEL org.opencontainers.image.description="Expenses tracker service"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=build /out/expenses-tracker /app/expenses-tracker

EXPOSE 8080

ENTRYPOINT ["/app/expenses-tracker"]
