# syntax=docker/dockerfile:1

FROM golang AS build

RUN mkdir /tmp/tri
WORKDIR /tmp/tri
COPY . .
RUN make

FROM alpine AS release

COPY --from=build /tmp/tri/bin/tri-linux-amd64 /usr/local/tri
ENTRYPOINT [ "/usr/local/tri", "-mount=/data", "-addr=0.0.0.0:3000" ]
