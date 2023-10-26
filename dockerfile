FROM gcr.io/distroless/static

ARG NAME
ENV name=${NAME}

COPY ${NAME} /app


ENTRYPOINT ["/app"]

