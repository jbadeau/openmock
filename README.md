[<img src="docs/logo.svg">](https://github.com/checkr/openmock)

<p align="center">
    <a href="https://goreportcard.com/report/github.com/checkr/openmock" target="_blank">
        <img src="https://goreportcard.com/badge/github.com/checkr/openmock">
    </a>
    <a href="https://circleci.com/gh/checkr/openmock" target="_blank">
        <img src="https://circleci.com/gh/checkr/openmock.svg?style=shield">
    </a>
    <a href="https://godoc.org/github.com/checkr/openmock" target="_blank">
        <img src="https://img.shields.io/badge/godoc-reference-green.svg">
    </a>
</p>

# OpenMock
OpenMock is a Go service that can mock services in integration tests, staging environment, or anywhere.
The goal is to simplify the process of writing mocks in various channels.
Currently it supports three channels:

- HTTP
- Kafka
- AMQP (e.g. RabbitMQ)

# Usage
Use it directly with the go cli: `om`.

```bash
$ go get github.com/checkr/openmock/cmd/om
$ OPENMOCK_TEMPLATES_DIR=./demo_templates om
```

Use it with docker.
```bash
$ docker run -it -p 9999:9999 -v $(pwd)/demo_templates:/data/templates checkr/openmock 
```

Test it.
```bash
$ curl localhost:9999/ping
```

Dependencies.
- HTTP (native supported, thanks to https://echo.labstack.com/)
- Kafka (optional)
  - To enable mocking kafka, set env `OPENMOCK_KAFKA_ENABLED=true`.
  - One can also config `OPENMOCK_KAFKA_CLIENT_ID` and `OPENMOCK_KAFKA_SEED_BROKERS`.
- AMQP (optional)
  - To enable mocking amqp, set env `OPENMOCK_AMQP_ENABLED=true`
  - One can also config `OPENMOCK_AMQP_URL`.

# OpenMock Templates
Templates are YAML files that describe the behavior of OpenMock.

### Templates Directory
You can put any number of `.yaml` or `.yml` files in a directory, and then point
environment variable `OPENMOCK_TEMPLATES_DIR` to it. OpenMock
will recursively (including subdirectories) load all the YAML files. For example:

```sh
# OPENMOCK_TEMPLATES_DIR=./demo_templates

./demo_templates
├── amqp.yaml
├── files
│   └── colors.json
├── http.yaml
├── jsonrpc.yaml
├── kafka.yaml
└── payload_from_file.yaml
```

### Templates Schema
```yaml
- key: name # the name of the mock

  ####################################################################
  ## Expect:
  ##   It represents the channel and condition for the mock.
  ##   For example, under what condition and from what channel should
  ##   we proceed the actions.
  ####################################################################
  expect:
    # Condition checks if we need to do the actions or not
    # It only proceeds if it evaluates to "true"
    condition: '{{.HTTPHeader.Get "X-Token" | eq "t1234"}}'
    # Use one (and only one) of the following channels - [http, kafka, amqp]
    http:
      method: GET
      path: /ping
    kafka:
      topic: hello_kafka_in
    amqp:
      exchange: exchange_1
      routing_key: key_in
      queue: key_in
  
  ####################################################################
  ## Actions:
  ##   Actions are a series of functions to run, which defines the
  ##   behaviors of the mock. Availabe actions are:
  ##     - publish_amqp
  ##     - publish_kafka
  ##     - redis
  ##     - reply_http
  ##     - send_http
  ##     - sleep
  ####################################################################
  actions:
    - publish_kafka:
        topic: hello_kafka_out
        payload: >
          {
            "kafka": "OK",
            "data": {}
          }
    - sleep:
        duration: 1s
    - reply_http:
        status_code: 200
        body: OK
        headers:
          Content-Type: text/html
```
### Dynamic templating
OpenMock leverages [https://golang.org/pkg/text/template/](https://golang.org/pkg/text/template/) to write dynamic templates. Specifically, it supports a lot of _Context_ and _Helper Functions_.

- Usage of `{{ expr }}`. One can put `{{ expr }}` inside three types of places:
  - `expect.condition`
  - `action.http.body`, `action.kafka.payload`, `action.amqp.payload`
  - `action.http.body_from_file`, `action.kafka.payload_from_file`, `action.amqp.payload_from_file` (`{{ expr }}` will be in the file)
- Use Context inside `{{ expr }}`.
  ```bash
  .HTTPHeader      # type: http.Header; example: {{.HTTPHeader.Get "X-Token"}}
  .HTTPBody        # type: string;      example: {{.HTTPBody}}
  .HTTPPath        # type: string;      example: {{.HTTPPath}}
  .HTTPQueryString # type: string;      example: {{.HTTPQueryString}}

  .KafkaTopic      # type: string;      example: {{.KafkaTopic}}
  .KafkaPayload    # type: string;      example: {{.KafkaPayload}}

  .AMQPExchange    # type: string;      example: {{.AMQPExchange}}
  .AMQPRoutingKey  # type: string;      example: {{.AMQPRoutingKey}}
  .AMQPQueue       # type: string;      example: {{.AMQPQueue}}
  .AMQPPayload     # type: string;      example: {{.AMQPPayload}}
  ```
- Use helper functions inside `{{ expr }}`. We recommend pipeline format (`|`) of the functions.
  ```bash
  # Supported functions defined in ./template_helper.go

    - jsonPath # doc: https://github.com/antchfx/xpath
    - xmlPath  # doc: https://github.com/antchfx/xpath
    - uuidv5   # uuid v5 sha1 hash
    - redisDo  # run redis commands. For example {{redisDo "RPUSH" "arr" "hi"}}
    - ...

  # Supported functions inherited from
  # https://github.com/Masterminds/sprig/blob/master/functions.go

    - replace
    - uuidv4
    - regexMatch
    - ...

  # Examples
  {{.HTTPHeader.Get "X-Token" | eq "t1234"}}
  {{.HTTPBody | jsonPath "user/first_name" | replace "A" "a" | uuidv5 }}
  {{.HTTPBody | xmlPath "node1/node2/node3"}}
  ```

### Example: Mock HTTP
```yaml
# demo_templates/http.yaml

- key: ping
  expect:
    http:
      method: GET
      path: /ping
  actions:
    - reply_http:
        status_code: 200
        body: OK
        headers:
          Content-Type: text/html

- key: header-token-ok
  expect:
    condition: '{{.HTTPHeader.Get "X-Token" | eq "t1234"}}'
    http:
      method: GET
      path: /token
  actions:
    - sleep:
        duration: 1s
    - reply_http:
        status_code: 200
        body: >
          { "hello": "you have a valid X-Token in the header" }

- key: header-token-not-ok
  expect:
    condition: '{{.HTTPHeader.Get "X-Token" | ne "t1234"}}'
    http:
      method: GET
      path: /token
  actions:
    - reply_http:
        status_code: 401
```

### Example: Mock Kafka
```yaml
# demo_templates/kafka.yaml

- key: test_kafka_1
  expect:
    kafka:
      topic: hello_kafka_in
  actions:
    - publish_kafka:
        topic: hello_kafka_out
        payload: >
          {
            "kafka": "OK",
            "data": {}
          }

- key: test_kafka_2
  expect:
    kafka:
      topic: hello_kafka_in
  actions:
    - publish_kafka:
        topic: hello_kafka_out
        payload_from_file: './files/colors.json' # the path is relative to OPENMOCK_TEMPLATES_DIR
```

### Example: Mock AMQP (e.g. RabbitMQ)
```yaml
# demo_templates/amqp.yaml

- key: test_amqp_1
  expect:
    amqp:
      exchange: exchange_1
      routing_key: key_in
      queue: key_in
  actions:
    - publish_amqp:
        exchange: exchange_1
        routing_key: key_out
        payload: >
          {
            "amqp": "OK",
            "data": {}
          }

- key: test_amqp_2
  expect:
    amqp:
      exchange: exchange_1
      routing_key: key_in
      queue: key_in
  actions:
    - publish_amqp:
        exchange: exchange_1
        routing_key: key_out
        payload_from_file: './files/colors.json'
```

### Example: Use Redis for stateful things (by default, OpenMock uses an in-memory miniredis)
```
# demo_templates/redis.yaml

- key: hello_redis
  expect:
    http:
      method: GET
      path: /test_redis
  actions:
    - redis:
      - '{{.HTTPHeader.Get "X-TOKEN" | redisDo "SET" "k1"}}'
      - '{{redisDo "RPUSH" "random" uuidv4}}'
      - '{{redisDo "RPUSH" "random" uuidv4}}'
      - '{{redisDo "RPUSH" "random" uuidv4}}'
    - reply_http:
        status_code: 200
        body: >
          {
            "k1": "{{redisDo "GET" "k1"}}",
            "randomStr": "{{redisDo "LRANGE" "random" 0 -1}}",
            "random": [
              {{ $arr := redisDo "LRANGE" "random" 0 -1 | splitList ";;" }}
              {{ range $i, $v := $arr }}
                {{if isLastIndex $i $arr}}
                  "{{$v}}"
                {{else}}
                  "{{$v}}",
                {{end}}
              {{end}}
            ]
          }

# To test
# curl localhost:9999/test_redis -H "X-TOKEN:t123"  | jq .
```

### Example: Send Webhooks
```
# demo_templates/webhook.yaml

- key: webhooks
  expect:
    http:
      method: GET
      path: /send_webhook_to_httpbin
  actions:
    - send_http:
        url: "https://httpbin.org/post"
        method: POST
        body: '{"hello": "world"}'
        headers:
          X-Token: t123
    - reply_http:
        status_code: 200
        body: 'webhooks sent'

# To test
# curl localhost:9999/send_webhook_to_httpbin
```

# Advanced pipeline functions
To enable advanced mocks, for example, your own encoding/decoding of the kafka messages,
one can develop by directly importing the `github.com/checkr/openmock` package.

For example:
```
package main

import "github.com/checkr/openmock"

func consumePipelineFunc(c *openmock.Context, in []byte) (out []byte, error) {
  return decode(in), nil
}

func publishPipelineFunc(c *openmock.Context, in []byte) (out []byte, error) {
  return encode(in), nil
}

func main() {
    om := openmock.OpenMock{
        KafkaConsumePipelineFunc: consumePipelineFunc,
        KafkaPublishPipelineFunc: publishPipelineFunc,
    }
    om.Start()
}
```
