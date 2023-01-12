FROM golang:1.19.4

COPY ./ /app

WORKDIR /app

RUN go build main.go

RUN chmod +x main

ENTRYPOINT [ "sh", "-c" ]

CMD [ "./main" ]

EXPOSE 8180