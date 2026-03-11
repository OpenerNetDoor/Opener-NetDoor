FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/node-agent ./cmd/node-agent

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/node-agent /node-agent
ENTRYPOINT ["/node-agent"]
