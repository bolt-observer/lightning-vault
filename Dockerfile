FROM golang:1.19 as app-builder

WORKDIR /go/src/app
COPY . .
#COPY .git .git
RUN make clean linux MULTIARCH=true && ls -ali ./release/macaroon_vault*linux*

FROM scratch
LABEL org.opencontainers.image.source=https://github.com/bolt-observer/macaroon_vault
LABEL org.opencontainers.image.description="Macaroon vault"
LABEL org.opencontainers.image.licenses=MIT

ENV PORT 1339
ENV AWS_DEFAULT_REGION us-east-1

COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=app-builder /go/src/app/release/macaroon_vault*linux* /macaroon_vault
VOLUME ["/tmp"]

USER 666
ENTRYPOINT ["/macaroon_vault", "-logtostderr=true"]

EXPOSE ${PORT}
