FROM alpine:3.6

# Allow delve to run on Alpine based containers.
# RUN apk add --no-cache libc6-compat

# Debugger
# COPY build/bin/dlv /root/
# RUN chmod +x /root/dlv

COPY build/_output/bin/kubedirector /root/kubedirector
RUN chmod +x /root/kubedirector

COPY build/_output/configcli.tgz /root/configcli.tgz

USER root
