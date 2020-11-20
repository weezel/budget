FROM golang:1.15 as builder
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
#RUN go install -v ./...
RUN apt-get update \
	&& apt-get clean \\
	&& rm -rf /var/lib/apt/lists/*
RUN make

FROM golang:1.15 as app
WORKDIR /app
COPY --from=builder --chown=1000:1000 /go/src/app/budget_linux_amd64 .
USER 1000:1000
CMD ["/app/budget_linux_amd64","/app/config/budget.toml"]
