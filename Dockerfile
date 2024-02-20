ARG GIT_DESCRIBE=0.0.0

FROM golang:1.22.0 as builder

WORKDIR /go/src

COPY go.mod go.sum ./

RUN go mod download

COPY ./*.go ./

RUN go mod vendor

ENV GIT_DESCRIBE=${GIT_DESCRIBE}

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags=-X=main.commit=${GIT_DESCRIBE} -mod=vendor -a -installsuffix cgo -o wrapper .


FROM debian:bullseye-slim AS final

ENV DEBIAN_FRONTEND="noninteractive"

RUN dpkg --add-architecture i386 && \
    apt-get update -y && \
    apt-get install -y --no-install-recommends \
	    apt-utils \
        lsb-release \
	    software-properties-common \
        ca-certificates \
        locales \
        wget \
        gnupg2 \
        cabextract \
        procps && \
    mkdir -pm755 /etc/apt/keyrings && \
    wget -O /etc/apt/keyrings/winehq-archive.key https://dl.winehq.org/wine-builds/winehq.key && \
    wget -NP /etc/apt/sources.list.d/ https://dl.winehq.org/wine-builds/debian/dists/bullseye/winehq-$(lsb_release -sc).sources && \
    apt-add-repository -y "deb http://deb.debian.org/debian $(lsb_release -sc) main contrib" && \
    apt-get update -y && \
    apt-get install -y --install-recommends \
        wine \
        wine32 \
        wine64 \
        libwine \
        libwine:i386 \
        fonts-wine \
        winetricks \ 
        winbind  && \
    apt-get -y autoremove && \
    rm -rf \
        /tmp/* \
        /var/tmp/* \
        /var/lib/apt/lists/*


ENV DATAPATH="/data"
ENV CONFIGPATH="/data/config.json"
ENV LOGPATH="/data/log/echovr.log"
ENV GAMEDATAPATH="/data/game"
ENV APPDATAPATH="/data/app"


VOLUME /echovr
COPY --link ./echovr/. /echovr/.

VOLUME /data
COPY ./data/. data/.

# Setup the application data volume
RUN mkdir -p /root/.wine/drive_c/users/root/Local\ Settings/Application\ Data && \
    ln -sf {$APPDATAPATH} /root/.wine/drive_c/users/root/Local\ Settings/Application\ Data/rad

RUN rm -f /echovr/_local/config.json && ln -s ${CONFIGPATH} /echovr/_local/config.json

RUN rm -rf /echovr/sourcedb/rad15/json/r14 && \
    ln -s ${GAMEDATAPATH} /echovr/sourcedb/rad15/json/r14

ENV WINEDEBUG=-all
ENV TERM=xterm

#USER nobody:nogroup

RUN winetricks -q winhttp && \
    wine wineboot -s

COPY --from=builder /go/src/wrapper /wrapper

WORKDIR /echovr/bin/win10

ENTRYPOINT ["/wrapper"]
