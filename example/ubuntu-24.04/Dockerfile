FROM ubuntu:24.04 AS build-env

# Set up dependencies
ENV PACKAGES="curl wget make git libc6-dev gcc g++ libudev-dev python3 ca-certificates"

# Install minimum necessary dependencies
RUN apt-get update \
    && apt-get install -y --no-install-recommends $PACKAGES \
    && update-ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install Go 1.22.12
RUN wget https://go.dev/dl/go1.22.12.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.12.linux-amd64.tar.gz && \
    ln -s /usr/local/go/bin/go /usr/bin/go && \
    rm go1.22.12.linux-amd64.tar.gz

# Set working directory for the build
WORKDIR /go/src/github.com/stratosnet/sds

COPY go.mod go.sum ./
RUN go mod download

# Add source files
COPY . .
RUN make update install

RUN cd relayer && make install


# Final image
FROM ubuntu:24.04

ENV WORK_DIR=/sds
ENV RUN_AS_USER=sds

# Install ca-certificates
RUN apt-get update \
    && apt-get install -y --no-install-recommends gosu ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && gosu nobody true

# Default non-root user ID, override with --build-arg UID=$(id -u)
ARG UID=2048
ARG GID=2048
ARG RUN_AS_USER=sds

RUN groupadd -g ${GID} ${RUN_AS_USER} 2>/dev/null || true && \
    useradd -u ${UID} -g ${GID} -m -d ${WORK_DIR} -s /bin/bash ${RUN_AS_USER} 2>/dev/null || true

WORKDIR ${WORK_DIR}

# Copy over binaries from the build-env
COPY --from=build-env /root/go/bin/ppd /usr/bin/
COPY example/ubuntu-24.04/entrypoint.sh /usr/bin/entrypoint.sh

RUN chmod +x /usr/bin/entrypoint.sh

ENTRYPOINT ["/usr/bin/entrypoint.sh"]
CMD ["ppd", "start"]

