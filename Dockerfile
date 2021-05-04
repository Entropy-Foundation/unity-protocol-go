FROM golang

WORKDIR /app

COPY ./ /go/src/github.com/user/unity/unity
WORKDIR /go/src/github.com/user/unity/unity

RUN go env
RUN go get ./
RUN go build

CMD unity