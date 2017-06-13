FROM golang:latest

VOLUME /policy
EXPOSE 5877

WORKDIR /go/src/app
COPY . .

WORKDIR /go/src/app/cmd/resourceful
RUN go get -v -d -u . && go install -v .

WORKDIR /policy
CMD ["/go/bin/resourceful", "guardian"]
