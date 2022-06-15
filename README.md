# Kafka test evaluation

This experiment tries to check how different Kafka Go clients behave with respect to batching
accumulation.

To run:

```
docker compose up
go run main.go
```

Main accepts the following arguments:

```
  -a    asynchronous writing
  -b string
        Kafka broker address (default "127.0.0.1:29092")
  -h    this help
  -l string
        producer lib: segmentio (kafka-go), sarama or confluent (default "segmentio")
  -mps int
        messages per second (default 500)
  -s int
        batch size (default 50)
  -t duration
        batch timeout (default 1s)
```

## Segmentio's Kafka-go

```
$ go run main.go   
messages/sec = 500      batchSize = 50  batchTimeout = 1s
2022/06/15 12:10:10 0.8 messages/s
2022/06/15 12:10:15 1.0 messages/s
2022/06/15 12:10:20 1.0 messages/s
```

```
$ go run main.go -a
messages/sec = 500      batchSize = 50  batchTimeout = 1s
2022/06/15 12:10:37 499.4 messages/s
2022/06/15 12:10:42 499.2 messages/s
```

## Sarama

```
$ go run main.go -sarama
messages/sec = 500      batchSize = 50  batchTimeout = 1s
2022/06/15 12:11:43 0.8 messages/s
2022/06/15 12:11:48 1.0 messages/s
2022/06/15 12:11:53 1.0 messages/s
2022/06/15 12:11:58 1.0 messages/s
2022/06/15 12:12:03 1.0 messages/s
2022/06/15 12:12:08 1.0 messages/s
```

```
$ go run main.go -sarama -a    
messages/sec = 500      batchSize = 50  batchTimeout = 1s
2022/06/15 12:14:20 497.4 messages/s
2022/06/15 12:14:25 499.4 messages/s
```

## Confluent

```
$ go run main.go -l confluent
messages/sec = 500      batchSize = 50  batchTimeout = 1s
2022/06/15 12:35:42 498.8 messages/s
2022/06/15 12:35:47 498.6 messages/s
2022/06/15 12:35:52 498.6 messages/s
```

(async omitted as results are the same)

```
$ go run main.go -l confluent -mps 5
messages/sec = 5        batchSize = 50  batchTimeout = 1s
%3|1655289637.693|FAIL|rdkafka#producer-1| [thrd:localhost:29092/1]: localhost:29092/1: Connect to ipv6#[::1]:29092 failed: Connection refused (after 2ms in state CONNECT)
2022/06/15 12:40:42 5.0 messages/s
```