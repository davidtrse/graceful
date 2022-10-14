up: docker-compose up -d
init: docker exec broker \
	kafka-topics --bootstrap-server broker:9092 \
             --create \
             --topic topic-test
.PHONY: up