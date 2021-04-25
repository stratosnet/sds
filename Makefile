update:
	go mod vendor

build_pp:
	go build -o ./target/ ./pp/

build_sp:
	go build -o ./target/ ./example/sp/src/sp.go

build_pp_example:
	docker rm -f sds-mysql1 sds-redis
	docker run -d --name sds-mysql1 -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql
	sleep 10 # otherwise mysql does not have time to start and the next command will fail
	# build tables in mysql
	docker cp ./example/database/sp_database.sql sds-mysql1:/home
	docker exec sds-mysql1 bash -c 'mysql -h 0.0.0.0 -u user1 -p111111 < /home/sp_database.sql'
	docker run -d --name sds-redis -p 6379:6379 redis
	go run ./example/pp_main/main.go

restart-db:
	docker rm -f sds-redis sds-mysql
	docker run --name sds-redis -d -p 6379:6379 redis
	docker run -d --name sds-mysql -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql

	sleep 10
	docker cp ./example/database/sp_database.sql sds-mysql:/home
	docker exec sds-mysql bash -c 'mysql -h 0.0.0.0 -u user1 -p111111 < /home/sp_database.sql'