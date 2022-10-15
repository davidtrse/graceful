up: docker-compose up -d
topic: docker exec broker \
	kafka-topics --bootstrap-server broker:9092 \
             --create \
             --topic topic-test
.PHONY: up topic