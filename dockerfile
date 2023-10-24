FROM gcr.io/distroless/static

ARG NAME

COPY ${NAME} /app
COPY ${name}-bin ${name}-bin 

COPY service-config.json service-config.json

ENTRYPOINT ["/app"]

