FROM alpine:3.11.2

VOLUME /simplex/inputs /simplex/outputs

RUN mkdir -p /scripts && \
    echo "cp /simplex/inputs/inputs.txt /simplex/outputs/outputs.txt" >/scripts/run.sh && \
    echo 'echo ${MY_ENV:-"+1"} >>/simplex/outputs/outputs.txt' >>/scripts/run.sh

WORKDIR /scripts

ENTRYPOINT ["sh", "run.sh"]
