FROM golang:1.26 AS build
WORKDIR /app
COPY . .
RUN go mod download && go build -o iptracker -v ./... && chmod +x iptracker

FROM scratch
COPY --from=build /app/iptracker .
CMD ["iptracker"]