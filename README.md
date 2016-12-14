# Moonlight

Moonlight is a kind of light produced by the
[Lighthouse](https://github.com/googlechrome/lighthouse).
It provides Lighthouse-As-A-Service: submit a URL, get results.

The service accepts the following parameters at `/audit` path:

Name       | Description
---------- | ------------
`url`      | Required. URL to analyze, including proto scheme.
`fmt`      | Output format. Detauls to `html`. Other values are: `json`, `pretty`.

For instance, to get a JSON response:

https://HOST/audit?url=https%3A%2F%2Fexample.com&fmt=json

## Design

      request url=https://...  +----------+                +-----------+
    -------------------------> |          |                |           |
                               |          | runs Chrome    |  Chrome   |
      response                 | frontend | ------------>  |           |
    <------------------------- |          |                +-----------+
                               +----------+                     ∧  |
                                  ∧  |                          |  |
                                  |  | execute lighthouse-cli   |  |
                           stdout |  | with requested url       |  ∇
                           stderr |  | and other parameters     | GET
                                  |  V                          | https://...
                        +----------------+                      |
                        |                |                      |
                        | lighthouse-cli | ---------------------+
                        |                |
                        +----------------+


* frontend is a simple HTTP server which forwards incoming requests
  to `lighthouse-cli` and responds back with the command result.
* Chrome is running in headless mode, using the headless shell, which is spun up on each request.
* lighthouse-cli executed as a shell command from the frontend server,
  connecting to a randomly selected debugging port to accommodate concurrent requests.

All of Moonlight functionality in the diagram above is encompassed in a Docker container,
which makes it highly portable and easy to deploy.

In fact, moonlight is running the docker container hosted by App
Engine Flex environment.

## Local development

You'll need to be able to build [Docker](https://www.docker.com) containers locally
and have [Go 1.6+](https://golang.org/doc/install) installed.

To build the container image execute:

    make

    # or, without make:
    GOOS=linux GOARCH=amd64 go build -o bin/server ./server/*.go
    curl -sSL -o bin/headless-shell.tar.gz \
      https://storage.googleapis.com/moonlight-files/headless-shell.tar.gz
    docker build --rm -t moonlight .

This will create a Docker image called `moonlight`. You can then run it as
follows:

    make run

    # or, without make:
    docker run -ti --rm -p 8080:8080 moonlight

Having that running, you should be able to access it locally. For instance, try
this:

    curl 'localhost:8080/audit?fmt=pretty&url=https://example.com'

## Deployment

To deploy to moonlight App Engine app, you'll need
[gcloud](https://cloud.google.com/sdk/gcloud/).

Once installed, use this to deploy:

    make deploy

    # or, without make:

    # first, tag `moonlight` image to point to a gcr.io repository
    docker tag -f moonlight gcr.io/google.com/moonlight/server
    # then, push it with:
    gcloud docker push gcr.io/google.com/moonlight/server
    # finally, deploy using gcloud
    gcloud app deploy --project=google.com:moonlight server/app.yaml \
      --image-url=gcr.io/google.com/moonlight/server

This may take a few minutes, but eventually you should have it running and accessible.
