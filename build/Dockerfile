FROM alpine:3.6
ENV OPERATOR=/usr/local/bin/kubedirector

# Allow delve to run on Alpine based containers.
# RUN apk add --no-cache libc6-compat

# Debugger
# COPY build/bin/dlv /root/
# RUN chmod +x /root/dlv

COPY build/_output/bin/kubedirector ${OPERATOR}
RUN chmod +x ${OPERATOR}

COPY build/_output/configcli.tgz /root/configcli.tgz

USER root
