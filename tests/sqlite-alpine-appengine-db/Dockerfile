FROM alpine:latest
ARG APPENGDBPATH=/data/app_engine/db
RUN mkdir -p ${APPENGDBPATH} && echo "HOLA" > /canary && mkdir -p ${APPENGDBPATH}/not.a.db
COPY test-apps-and-device.db ${APPENGDBPATH}/platformbox.db
ENTRYPOINT ["/bin/sh", "-c", "i=60; while [ $i -ne 0 ]; do sleep 1; i=$(($i-1)); done"]
