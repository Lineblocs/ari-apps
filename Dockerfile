# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:1.17.1

# Add Maintainer Info
LABEL maintainer="Nadir Hamid <matrix.nad@gmail.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

RUN apt-get -y update && apt-get install -y bash netdiscover
# Copy go mod and sum files
COPY go.mod go.sum ./

ADD keys/key /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa
ADD .gitconfig /root/.gitconfig
#RUN echo "Host bitbucket.org\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
#RUN git config --global url.ssh://git@bitbucket.org/.insteadOf https://bitbucket.org/
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main main.go

# Expose port 80 to the outside world (used for GRPC)
EXPOSE 8018

RUN ls -a /app/
# Command to run the executable
ENTRYPOINT ["/bin/bash", "./entrypoint.sh"]