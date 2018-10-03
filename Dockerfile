FROM golang:1.11-alpine

COPY . /go/src/github.com/cyverse-de/de-webhooks
ENV CGO_ENABLED=0
RUN go install -v github.com/cyverse-de/de-webhooks

ENTRYPOINT ["de-webhooks"]

ARG git_commit=unknown
ARG version="2.9.0"
ARG descriptive_version=unknown

LABEL org.cyverse.git-ref="$git_commit"
LABEL org.cyverse.version="$version"
LABEL org.cyverse.descriptive-version="$descriptive_version"

LABEL org.label-schema.vcs-ref="$git_commit"
LABEL org.label-schema.vcs-url="https://github.com/cyverse-de/de-webhooks"
LABEL org.label-schema.version="$descriptive_version"
