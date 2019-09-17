FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV USER_UID=1001 \
    USER_NAME=kubedirector \
    USER_HOME=/home/kubedirector

# Support ps and killall, for redeploy.
RUN microdnf update -y && rm -rf /var/cache/yum
RUN microdnf -y install --nodocs psmisc procps-ng \
    && microdnf clean all

# For OpenShift support, the runtime user will need to be able to self-insert
# in /etc/passwd. We can't add the user during image build because on OpenShift
# we'll get a random user ID at runtime (rather than USER_UID).
RUN chmod g=u /etc/passwd

# Install entrypoint script. This will create the user passwd entry and
# launch the operator.
COPY build/bin/entrypoint /usr/local/bin/entrypoint
RUN chown 0:0 /usr/local/bin/entrypoint && \
    chmod u=rwx,go=rx /usr/local/bin/entrypoint

# Set up the kubedirector user files, including the operator binary. For
# OpenShift support, make sure they are usable by any user in the root group.
RUN mkdir -p ${USER_HOME} && \
    chown ${USER_UID}:0 ${USER_HOME} && \
    chmod ug=rwx ${USER_HOME}
COPY build/_output/bin/kubedirector ${USER_HOME}/kubedirector
COPY build/_output/configcli.tgz ${USER_HOME}/configcli.tgz
RUN chown ${USER_UID}:0 ${USER_HOME}/kubedirector && \
    chmod ug=rwx ${USER_HOME}/kubedirector && \
    chown ${USER_UID}:0 ${USER_HOME}/configcli.tgz && \
    chmod ug=rw ${USER_HOME}/configcli.tgz

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:0
