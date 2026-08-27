FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /botdo .

FROM alpine:3.20
RUN addgroup -S botdo && adduser -S botdo -G botdo
WORKDIR /data
COPY --from=build /botdo /usr/local/bin/botdo
RUN chown botdo:botdo /data
USER botdo
ENV BOTDO_ADDR=:8080 BOTDO_DATA=/data/botdo.json
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["botdo"]
CMD ["--no-agent"]
