ARG BUILTBY
ARG DATE
ARG COMMIT
ARG VERSION

FROM golang:1.13-alpine AS build
ARG BUILTBY
ARG DATE
ARG COMMIT
ARG VERSION

WORKDIR /home
COPY src src
RUN cd src && go build -o ../vaultlink -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE -X main.builtBy=$BUILTBY"

FROM alpine:3.10
COPY --from=build /home/vaultlink vaultlink
CMD ["./vaultlink"]
