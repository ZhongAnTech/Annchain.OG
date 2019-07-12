import threading, logging, time
import multiprocessing
import json

from kafka import KafkaConsumer, KafkaProducer


class Producer(threading.Thread):
    def __init__(self):
        threading.Thread.__init__(self)
        self.stop_event = threading.Event()

    def stop(self):
        self.stop_event.set()

    def run(self):
        producer = KafkaProducer(bootstrap_servers='10.253.11.192:9092')

        while not self.stop_event.is_set():
            for i in range  (1000000):
                d = {
                     "thread": "http-nio-8080-exec-5",
                    "level": "INFO",
                    "loggerName": "auditing",
                    "message": "TTT",
                    "endOfBatch": False,
                    "loggerFqcn": "org.apache.logging.slf4j.Log4jLogger",
                    "instant": {
                        "epochSecond": 1561375556,
                        "nanoOfSecond": 447000000
                    },
                    "contextMap": {
                        "id": "122",
                        "user": "XXX"
                    },
                    "threadId": i,
                    "threadPriority": 5
                }
                ss = json.dumps(d)
                #ss += '\0'
                #producer.send('tech-tech-anlink-web-gateway-201907101551', bytes(ss, 'utf-8'))
                producer.send('tech-tech-anlink-web-gateway-201907101551', bytes(str(i), 'utf-8'))


                #time.sleep(0.01)
                #print(ss)
            print("end")
        producer.close()


class Consumer(multiprocessing.Process):
    def __init__(self):
        multiprocessing.Process.__init__(self)
        self.stop_event = multiprocessing.Event()

    def stop(self):
        self.stop_event.set()

    def run(self):
        consumer = KafkaConsumer(bootstrap_servers='localhost:9092',
                                 auto_offset_reset='earliest',
                                 consumer_timeout_ms=1000)
        consumer.subscribe(['my-topic'])

        while not self.stop_event.is_set():
            for message in consumer:
                print(message)
                if self.stop_event.is_set():
                    break

        consumer.close()


def main():
    tasks = [
        Producer(),
        #Consumer()
    ]

    for t in tasks:
        t.start()

    time.sleep(10)

    for task in tasks:
        task.stop()

    for task in tasks:
        task.join()


if __name__ == "__main__":
    logging.basicConfig(
        format='%(asctime)s.%(msecs)s:%(name)s:%(thread)d:%(levelname)s:%(process)d:%(message)s',
        level=logging.INFO
    )
    main()