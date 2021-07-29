# docker run -d --name sds-mysql -e MYSQL_ROOT_PASSWORD=111111 -e MYSQL_DATABASE=sds -e MYSQL_USER=user1 -e MYSQL_PASSWORD=111111 -p 3306:3306 mysql
USE sds;

CREATE TABLE `file`
(
    `id`                int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
    `name`              varchar(128) NOT NULL DEFAULT '',
    `hash`              char(64)     NOT NULL DEFAULT '',
    `size`              bigint(20) unsigned NOT NULL DEFAULT '0',
    `slice_num`         int(10) unsigned NOT NULL DEFAULT '0',
    `state`             tinyint(3) unsigned NOT NULL DEFAULT '0',
    `download`          int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'download count',
    `time`              int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'upload time',
    `is_cover`          tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT 'is cover',
    `is_video_stream`   tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT 'is video steam',
    PRIMARY KEY (`id`),
    UNIQUE KEY `IDX_HASH` (`hash`) USING HASH,
    KEY         `IDX_NAME` (`name`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

CREATE TABLE pp
(
    id              int unsigned     NOT NULL AUTO_INCREMENT COMMENT 'Id of pp' PRIMARY KEY,
    p2p_address     char(255)        NOT NULL DEFAULT '',
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
    active          tinyint          NOT NULL DEFAULT '0',
    UNIQUE KEY IDX_P2P_ADDRESS (p2p_address) USING HASH
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
    p2p_address     varchar(256) null,
    wallet_address  varchar(256) null,
    network_Address varchar(256) null,
    UNIQUE KEY IDX_P2P_ADDRESS (p2p_address) USING HASH
) ENGINE = InnoDB
  DEFAULT CHARSET = UTF8MB4;

CREATE TABLE `transfer_record`
(
    `id`                   int(10) unsigned    NOT NULL AUTO_INCREMENT COMMENT 'ID' PRIMARY KEY,
    `slice_hash`           char(64)            NOT NULL DEFAULT '',
    `slice_size`           bigint unsigned     NOT NULL DEFAULT '0',
    `transfer_cer`         char(64)            NOT NULL DEFAULT '',
    `from_p2p_address`     char(255)           NOT NULL DEFAULT '' COMMENT 'origin PP P2P key address',
    `from_wallet_address`  char(42)            NOT NULL DEFAULT '' COMMENT 'origin PP wallet address',
    `from_network_address` varchar(32)         NOT NULL DEFAULT '' COMMENT 'origin PP network address',
    `to_p2p_address`       char(255)           NOT NULL DEFAULT '' COMMENT 'target PP P2P key address',
    `to_wallet_address`    char(42)            NOT NULL DEFAULT '' COMMENT 'target PP wallet address',
    `to_network_address`   varchar(32)         NOT NULL DEFAULT '' COMMENT 'target network address',
    `status`               tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:success,1:waiting,2:pending,3:error',
    `time`                 int(10) unsigned    NOT NULL DEFAULT '0' COMMENT 'transfer finish time',
    KEY `IDX_TRANSFER_CER` (`transfer_cer`) USING HASH
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = UTF8MB4;

create table user_has_file
(
    file_hash      varchar(256) null,
    wallet_address varchar(42) null
);

create table user_invite
(
    invitation_code varchar(256) null,
    wallet_address  varchar(42) null,
    times           int          null
);

CREATE TABLE `user_directory`
(
    `dir_hash`       char(64)         NOT NULL DEFAULT '',
    `wallet_address` char(42)                 DEFAULT '',
    `path`           varchar(512)     NOT NULL DEFAULT '',
    `time`           int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'creation time',
    PRIMARY KEY (`dir_hash`),
    UNIQUE KEY `IDX_WALLET_ADDRESS_PATH` (`wallet_address`,`path`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `user_directory_map_file`
(
    `dir_hash`     char(64)  NOT NULL DEFAULT '' COMMENT 'directory hash',
    `file_hash`    char(64)  NOT NULL DEFAULT '' COMMENT 'file hash',
    `owner_wallet` char(42) NOT NULL DEFAULT '' COMMENT 'owner wallet address',
    PRIMARY KEY (`dir_hash`, `file_hash`) USING HASH,
    KEY         `IDX_WALLET_ADDRESS` (`owner_wallet`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `file_slice`
(
    `id`                 int(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `file_hash`          char(64) NOT NULL DEFAULT '',
    `slice_hash`         char(64) NOT NULL DEFAULT '',
    `slice_size`         bigint(20) NOT NULL DEFAULT '0',
    `slice_number`       int(10) unsigned NOT NULL DEFAULT '1',
    `slice_offset_start` bigint(20) NOT NULL DEFAULT '0',
    `slice_offset_end`   bigint(20) NOT NULL DEFAULT '0',
    `status`             tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:success,1:pending,2:error',
    `task_id`            char(64) NOT NULL DEFAULT '',
    `time`               int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'upload time',
    PRIMARY KEY (`id`),
    KEY                  `IDX_FILE_HASH` (`file_hash`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;

CREATE TABLE `file_slice_storage`
(
    `slice_hash`      char(64)    NOT NULL DEFAULT '',
    `p2p_address`     char(255)   NOT NULL DEFAULT '' COMMENT 'storage PP P2P key address',
    `network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'storage PP network address',
    PRIMARY KEY (`slice_hash`, `p2p_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `file_download` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  `to_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'downloader wallet address',
  `task_id` char(64) NOT NULL DEFAULT '' ,
  `time` int(11) NOT NULL DEFAULT '0' COMMENT 'download time',
  PRIMARY KEY (`id`),
  KEY `IDX_FILE_HASH` (`file_hash`) USING HASH,
  KEY `IDX_TASK_ID` (`task_id`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `file_slice_download` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `slice_hash` char(64) NOT NULL DEFAULT '' ,
  `from_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'PP wallet address',
  `to_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'P wallet address',
  `status` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:success,1:pending,2:error',
  `task_id` char(64) NOT NULL DEFAULT '' ,
  `time` int(11) NOT NULL DEFAULT '0' COMMENT 'download time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;

CREATE TABLE traffic (
  id                      int(10) unsigned    NOT NULL AUTO_INCREMENT COMMENT 'ID',
  provider_p2p_address    char(255)           NOT NULL DEFAULT '' COMMENT 'PP P2P address',
  provider_wallet_address char(42)            NOT NULL DEFAULT '' COMMENT 'PP wallet address',
  consumer_wallet_address char(42)            NOT NULL DEFAULT '' COMMENT 'P wallet address',
  task_id                 char(64)            NOT NULL DEFAULT '' ,
  task_type               tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:upload,1:download,2:transfer',
  volume                  bigint(20)          NOT NULL DEFAULT '0' ,
  delivery_time           int(11)             NOT NULL DEFAULT '0' COMMENT 'delivery time',
  response_time           int(11)             NOT NULL DEFAULT '0' COMMENT 'response time',
  PRIMARY KEY (id)
) ENGINE=InnoDB  
DEFAULT CHARSET=utf8;

CREATE TABLE variables (
    name  varchar(64)  NOT NULL DEFAULT '',
    value varchar(256) NOT NULL DEFAULT '',
    PRIMARY KEY (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE user_ozone (
    wallet_address varchar(42) NOT NULL DEFAULT '',
    available_uoz  bigint      NOT NULL DEFAULT '0',
    PRIMARY KEY (wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;