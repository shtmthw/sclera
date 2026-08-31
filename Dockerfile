FROM golang:1.26.6 AS builder 
# uses the go ecosystem image to compile my server

WORKDIR /Sclera
# makes a folder called sclera, which it will write on, withing the container of the go ecosys

COPY go.mod go.sum ./
# copies the mentioned files from the current dir its sitting on thus ./ is being used

RUN go mod download
# downlaods the dependencies within the go.mod files to the containers Go module cache

COPY . .
# copies the whole server (the original /Sclera) into the go ecosys helded on linux 

RUN go build -o main ./cmd/server
#builds the server using the main.go file in ./cmd/server and puts the binary into the folder /Sclera 
#so the container now has /Sclera/main, none of the other tools that compiles the server presists.


FROM debian:bookworm-slim
#and here starts the new image, a lightweight debian linux distro


RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
    
# installs the CA certificate bundle into /etc/ssl/certs — bookworm-slim
# ships with nothing here by default, so any outbound HTTPS call (Resend,
# any other API) has no root certificates to verify the server's cert
# against and fails closed with an x509 "unknown authority" error.
# --no-install-recommends keeps this from dragging in extra unneeded packages,
# and the rm afterward drops apt's package index so it doesn't bloat the image.


WORKDIR /Sclera
#makes the Sclera folder within it

COPY --from=builder /Sclera/main .
COPY --from=builder /Sclera/tempFrontend .

#this takes the /Sclera/main . (all) from the build stage named builder (who knew...) and pastes into the /Sclera, so the current builder has
# /Sclera/main

CMD ["./main"]
#runs the /Sclera/main binary file and starts the server