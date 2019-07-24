# building
FROM golang:1.12-alpine

# installing deps
RUN apk update \
  && apk upgrade \
  && apk add --no-cache bash git openssh

# misc
ENV APP_PROJECT github.com/sotah-inc/steamwheedle-cartel-run-download-auctions/app

# working dir
WORKDIR /srv/app

# copying in source
COPY ./app /srv/app
RUN go mod download

# building the project
RUN go build -o /go/bin/app .