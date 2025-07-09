FROM alpine:latest
ARG APPENGDBPATH
ARG PLATFORMBOXDBNAME
RUN mkdir -p ${APPENGDBPATH} && echo "HOLA" > /canary && mkdir -p ${APPENGDBPATH}/not.a.db
COPY test-apps-and-device.db ${APPENGDBPATH}/${PLATFORMBOXDBNAME}
ENTRYPOINT ["/bin/sh", "-c", "while true; do sleep 1; done"]
