FROM golang:1.13-alpine AS build
WORKDIR /home
COPY src src
RUN cd src && go build -o ../vaultlink

FROM alpine:3.10
COPY --from=build /home/vaultlink vaultlink
CMD ["./vaultlink"]
