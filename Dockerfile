FROM golang:1.14.2 AS builder

# Build arguments
ARG binary_name=markdown2confluence
    # See ./sample-data/go-os-arch.csv for a table of OS & Architecture for your base image
ARG target_os=linux
ARG target_arch=amd64

# Build the server Binary
WORKDIR /app
#WORKDIR /go/src/${GIT_SERVER}/${GIT_ORG}/${GIT_REPO}

COPY go.mod .
COPY go.sum .

RUN go mod download

# Seems duplicative, and ideally not needed
COPY . .

RUN rm -rf /app/build
RUN CGO_ENABLED=0 GOOS=${target_os} GOARCH=${target_arch} go build -a -o /app/build/${binary_name} main.go

RUN ls /app

#-----------------------------------------------------------------------------------------------------------------------

FROM centos:7

LABEL author="Benjamin Smith"
COPY --from=builder /app/build/markdown2confluence /usr/bin/markdown2confluence
RUN ["chmod", "-R", "+x", "/usr/bin/markdown2confluence"]

ENTRYPOINT ["tail", "-f", "/dev/null"]
