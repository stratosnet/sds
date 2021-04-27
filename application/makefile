clean-docker-images:
	docker rm -f sds-mysql sds-redis

run-mysql:
	docker run -d --name sds-mysql -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql

# mysql needs time to start and the next command will fail
sleep-10:
	sleep 10

# build tables in mysql
setup-mysql:
	docker cp ./sp/database/sp_database.sql sds-mysql:/home
	docker exec sds-mysql bash -c 'mysql -h 0.0.0.0 -u user1 -p111111 < /home/sp_database.sql'

run-redis:
	docker run --name sds-redis -d -p 6379:6379 redis

setup-databases: clean-docker-images run-mysql run-redis sleep-10 setup-mysql

build:
	go build -o sp/spServer sp/spServer.go

run-sp-server: setup-databases
	cd sp && ./spServer

# Then to run PP, cd into application/pp/node1 and node2
# go run pp_cmd.go

# type "rp" command to register PP node
# type "start" command to mine

# note there will be some getting pp node failed message, ignore it and just type those command and enter

