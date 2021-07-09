# kafka topic exporter

This exporter is used to consume custom monitoring data from Kafka topic, and then export out with Prometheus metric format.

## Use it

You need to clone this repo, then compile it by yourself.

```
  git clone https://github.com/destinyyu/kafka-topic-exporter
  cd kafka-topic-exporter
  go build
```

If you want run it into container, need an additinal step to build an image.

```
  docker build -t <image> .
```

### Flags

```
    -brokers string
        Kafka brokers
    -topics string
        consumer topics
    -drop_labels string
        labels which will be dropped
    -expiration int
        expired seconds for metrics, default 0 means never expiration
    -export_timestamp
        export timestamp from custom data, default use current time
    -group string
        consumer group
    -level string
        Logger level (default "info")
    -path string
        Path to export metrics on (default "/metrics")
    -port int
        Port to export metrics on (default 9000)
```

## Data Format

Each record in the Kafka topic should be the following format.

```
    {
        "metric": "test",           //metric name
        "value": 1.0,               //float64
        "valueType": "GAUGE",       //only support 'GAUGE' or 'COUNTER'
        "timestamp": 1625068800,    //optional, int64
        "labelname_1": "labelvalue_1",
        "labelname_2": "labelvalue_2",
        "labelname_3": "labelvalue_3",
        ...
    }
```

The exporter will export out like this

```
    <metric>{labelname_1="labelvalue_1",labelname_2="labelvalue_2"...} <value>
```
