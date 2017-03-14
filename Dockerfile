FROM golang
ADD . /go/src/github.com/coreos/ignition
RUN go install github.com/coreos/ignition/cmd/ignition
ENTRYPOINT ["ignition"]
