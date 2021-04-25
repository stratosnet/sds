# docker run -d --name sds-mysql -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql
USE sds;

create table file
(
    size int          null,
    hash varchar(256) null
);

CREATE TABLE pp
(
    id              int unsigned     NOT NULL AUTO_INCREMENT COMMENT 'Id of pp' PRIMARY KEY,
    wallet_address  char(42)         NOT NULL DEFAULT '',
    network_address varchar(32)      NOT NULL DEFAULT '',
    disk_size       bigint unsigned  NOT NULL DEFAULT '0',
    free_disk       bigint unsigned  NOT NULL DEFAULT '0',
    memory_size     bigint unsigned  NOT NULL DEFAULT '0',
    os_and_ver      varchar(128)     NOT NULL DEFAULT '',
    cpu_info        varchar(64)      NOT NULL DEFAULT '',
    mac_address     varchar(17)      NOT NULL DEFAULT '',
    version         int unsigned     NOT NULL DEFAULT '0',
    pub_key         varchar(1000)    NOT NULL DEFAULT '',
    state           tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:offline,1:online',
    UNIQUE KEY IDX_WALLET_ADDRESS (wallet_address) USING HASH
) ENGINE = InnoDB
  DEFAULT CHARSET = UTF8MB4;

create table user
(
    id              int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Id of user' PRIMARY KEY,
    name            varchar(256) null,
    register_time   int          null,
    invitation_code varchar(256) null,
    disk_size       int          null,
    capacity        int          null,
    be_invited      tinyint(1)   null,
    last_login_time int          null,
    login_times     int          null,
    belong          varchar(256) null,
    free_disk       int          null,
    puk             varchar(256) null,
    used_capacity   int          null,
    is_upgrade      tinyint(1)   null,
    is_pp           tinyint(1)   null,
    wallet_address  varchar(256) null,
    network_Address varchar(256) null,
    UNIQUE KEY IDX_WALLET_ADDRESS (wallet_address) USING HASH
) ENGINE = InnoDB
  DEFAULT CHARSET = UTF8MB4;

CREATE TABLE `transfer_record`
(
    `id`                   int(10) unsigned    NOT NULL AUTO_INCREMENT COMMENT 'ID' PRIMARY KEY,
    `file_slice_id`        int(10) unsigned    NOT NULL DEFAULT '0',
    `transfer_cer`         char(64)            NOT NULL DEFAULT '',
    `from_wallet_address`  char(42)            NOT NULL DEFAULT '' COMMENT 'origin PP wallet address',
    `to_wallet_address`    char(42)            NOT NULL DEFAULT '' COMMENT 'target PP wallet address',
    `from_network_address` varchar(32)         NOT NULL DEFAULT '' COMMENT 'origin PP network address',
    `to_network_address`   varchar(32)         NOT NULL DEFAULT '' COMMENT 'target network address',
    `status`               tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:success,1:waiting,2:pending,3:error',
    `time`                 int(10) unsigned    NOT NULL DEFAULT '0' COMMENT 'transfer finish time',
    KEY `IDX_FILE_SLICE_ID` (`file_slice_id`) USING BTREE,
    KEY `IDX_TRANSFER_CER` (`transfer_cer`) USING HASH
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = UTF8MB4;

create table user_has_file
(
    file_hash      varchar(256) null,
    wallet_address varchar(256) null
);

create table user_invite
(
    invitation_code varchar(256) null,
    wallet_address  varchar(256) null,
    times           int          null
);

