build_pp_example:
	docker rm -f sds-mysql1 sds-redis
	docker run -d --name sds-mysql1 -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql
	sleep 10 # otherwise mysql does not have time to start and the next command will fail
	# build tables in mysql
	docker cp ./database/sp_database.sql sds-mysql1:/home
	docker exec sds-mysql1 bash -c 'mysql -h 0.0.0.0 -u user1 -p111111 < /home/sp_database.sql'
	docker run -d --name sds-redis -p 6379:6379 redis
	go run main.go
