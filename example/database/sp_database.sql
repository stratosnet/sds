# docker run -d --name sds-mysql -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql
create table file
(
    size int null,
    hash varchar(256) null
);

create table pp
(
    ID int null,
    wallet_address varchar(256) null,
    network_Address varchar(256) null,
    pub_key varchar(256) null,
    state tinyint(1) null
);

create table user
(
    name varchar(256) null,
    register_time int null,
    invitation_code varchar(256) null,
    disk_size int null,
    capacity int null,
    be_invited tinyint(1) null,
    last_login_time int null,
    login_times int null,
    belong varchar(256) null,
    free_disk int null,
    puk varchar(256) null,
    used_capacity int null,
    is_upgrade tinyint(1) null,
    is_pp tinyint(1) null,
    wallet_address varchar(256) null,
    network_Address varchar(256) null
);

create table user_has_file
(
    file_hash varchar(256) null,
    wallet_address varchar(256) null
);

create table user_invite
(
    invitation_code varchar(256) null,
    wallet_address varchar(256) null,
    times int null
);

