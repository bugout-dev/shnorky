FROM alpine:3.11.2

VOLUME /shnorky

RUN mkdir -p /scripts && \
    echo "set -e" >/scripts/run.sh && \
    echo "cat /shnorky/inputs.txt >>/shnorky/outputs.txt" >>/scripts/run.sh && \
    echo 'echo ${MY_ENV:-"+1"} >>/shnorky/outputs.txt' >>/scripts/run.sh

WORKDIR /scripts

ENTRYPOINT ["sh", "run.sh"]
