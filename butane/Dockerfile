FROM quay.io/fedora/fedora:44 AS builder
RUN dnf install -y golang git-core
RUN mkdir /ignition
COPY . /ignition
WORKDIR /ignition
RUN ./butane/build_for_container

FROM quay.io/fedora/fedora-minimal:44
COPY --from=builder /ignition/bin/container/butane /usr/local/bin/butane
ENTRYPOINT ["/usr/local/bin/butane"]
