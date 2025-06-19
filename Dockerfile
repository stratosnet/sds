FROM ubuntu:20.04 AS build-env

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
FROM ubuntu:20.04

ENV WORK_DIR=/sds
ENV RUN_AS_USER=sds

# Install ca-certificates
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ARG uid=2048
ARG gid=2048

RUN addgroup --gid $gid "$RUN_AS_USER" && \
    useradd --uid $uid --gid $gid --home-dir "$WORK_DIR" --create-home --shell /bin/bash "$RUN_AS_USER"

WORKDIR $WORK_DIR

# Copy over binaries from the build-env
COPY --from=build-env /root/go/bin/ppd /root/go/bin/relayd /usr/bin/

COPY entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
CMD ["ppd start"]

