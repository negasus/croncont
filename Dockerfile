FROM golang:1.15 AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOFLAGS="-mod=vendor"

ARG version="undefined"

WORKDIR /build/croncont

ADD . /build/croncont

RUN go build -o /croncont -ldflags "-X main.version=${version} -s -w"  ./cmd/croncont

# -----

FROM  debian:stretch-slim
COPY --from=build /croncont /

RUN apt-get update \
     && apt-get install -y --no-install-recommends ca-certificates

RUN update-ca-certificates

ENTRYPOINT ["/croncont"]
