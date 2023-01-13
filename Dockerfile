FROM golang:1.19 as app-builder

WORKDIR /go/src/app
COPY . .
#COPY .git .git
RUN make clean linux MULTIARCH=true && ls -ali ./release/lightning-vault*linux*

FROM scratch
LABEL org.opencontainers.image.source=https://github.com/bolt-observer/lightning-vault
LABEL org.opencontainers.image.description="Lightning vault"
LABEL org.opencontainers.image.licenses=MIT

ENV PORT 1339
ENV AWS_DEFAULT_REGION us-east-1

COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=app-builder /go/src/app/release/lightning-vault*linux* /lightning-vault
VOLUME ["/tmp"]

USER 666
ENTRYPOINT ["/lightning-vault", "-logtostderr=true"]

EXPOSE ${PORT}
