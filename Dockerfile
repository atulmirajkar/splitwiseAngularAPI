FROM golang:alpine as builder

#install git
RUN apk add --update --no-cache git

#add local src folder to image src folder
ADD . $GOPATH/src/splitwiseAngularAPI/

#first install 
WORKDIR $GOPATH/src/splitwiseAngularAPI/

#get dependencies
RUN go get ./...

#run build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/splitwiseAngularAPI

#build a small image
FROM scratch

#install ca certificates necessary for smtp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


#copy static exec
WORKDIR /go/bin
COPY --from=builder /go/bin/splitwiseAngularAPI .

#copy static html and config files
COPY --from=builder /go/src/splitwiseAngularAPI/config.json .


#entrypoint
#ENTRYPOINT ["/go/bin/splitwiseAngularAPI","-config=configEncrypted.txt","-log=./data/passwordserver.log"]
ENTRYPOINT ["/go/bin/splitwiseAngularAPI"]

#expose port
EXPOSE 9094

#Build
#docker build -t expensegoapi . 

#run
#docker run -p 9094:9094 --name expensegoapi

