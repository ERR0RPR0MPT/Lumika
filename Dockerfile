FROM golang:latest
EXPOSE 7860

WORKDIR /temp

RUN apt update && \
        apt install ffmpeg git -y && \
        mkdir -p /app && \
        git clone https://github.com/ERR0RPR0MPT/Lumika.git && \
        cd Lumika && \
        go build -o /app/lumika . && \
        cd /app && \
        rm -rf /temp && \
        chmod 777 /app && \
        chmod a+x /app/lumika && \
        sleep 2

WORKDIR /app

ENTRYPOINT ["/app/lumika"]
