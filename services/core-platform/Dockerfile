FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/core-platform ./cmd/core-platform

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/core-platform /core-platform
EXPOSE 8081
ENTRYPOINT ["/core-platform"]
