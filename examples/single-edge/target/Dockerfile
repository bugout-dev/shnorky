FROM python:3.8.1-alpine3.11

RUN mkdir -p /simplex/inputs /simplex/outputs

VOLUME /simplex/inputs /simplex/outputs

WORKDIR /usr/src/app

COPY run.py .

ENTRYPOINT [ "python", "run.py" ]
