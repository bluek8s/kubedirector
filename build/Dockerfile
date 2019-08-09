FROM registry.access.redhat.com/ubi7/ubi-minimal:latest
ENV OPERATOR=/usr/local/bin/kubedirector
# ENV DEBUGGER=/usr/local/bin/dlv

RUN microdnf update -y && rm -rf /var/cache/yum

RUN microdnf -y install --nodocs psmisc procps-ng \
    && microdnf clean all

# Allow delve to run on Alpine based containers.
# RUN apk add --no-cache libc6-compat

# Debugger
# COPY build/bin/dlv ${DEBUGGER}
# RUN chmod +x ${DEBUGGER}

COPY build/_output/bin/kubedirector ${OPERATOR}
RUN chmod +x ${OPERATOR}

COPY build/_output/configcli.tgz /root/configcli.tgz

USER root
