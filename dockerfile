FROM alpine:3.18

ARG NAME
ENV name=${NAME}

COPY ${NAME} /app


ENTRYPOINT ["/app"]

