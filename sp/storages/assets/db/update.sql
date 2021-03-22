-- sp version 1.2

alter table download_record add column task_id char(64) not null default '' comment 'task id' after status;
alter table file_slice add column task_id char(64) not null default '' comment 'task ID' after status;
alter table file add column `state` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT 'file state' after owner;

update file set state = 1 where state = 0;

CREATE VIEW `v_upload_file` AS select `result`.`name` AS `name`,date_format(from_unixtime(`result`.`time`),'%Y/%m/%d') AS `upload_time` from (select `f`.`name` AS `name`,`fs`.`file_hash` AS `file_hash`,`fs`.`time` AS `time` from (`spacebook_sp`.`file_slice` `fs` left join `spacebook_sp`.`file` `f` on((`fs`.`file_hash` = `f`.`hash`))) group by `fs`.`file_hash`) `result`;
CREATE VIEW `v_upload_statistics` AS select count(*) AS `upload_total`,date_format(from_unixtime(`result`.`time`),'%Y/%m/%d') AS `time` from (select `spacebook_sp`.`file_slice`.`file_hash` AS `file_hash`,`spacebook_sp`.`file_slice`.`time` AS `time` from `spacebook_sp`.`file_slice` group by `spacebook_sp`.`file_slice`.`file_hash`) `result` group by date_format(from_unixtime(`result`.`time`),'%Y%m%d');
CREATE VIEW `v_download_statistics` AS select `f`.`name` AS `name`,count(*) AS `download_times` from ((`download_record` `dr` left join `file_slice` `fs` on((`dr`.`file_slice_id` = `fs`.`id`))) left join `file` `f` on((`f`.`hash` = `fs`.`file_hash`))) group by `f`.`id` order by `download_times` desc;

-- sp version 1.3
CREATE TABLE `user` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `is_pp` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:no，1:yes',
  `belong` char(42) NOT NULL DEFAULT '' COMMENT 'parent wallet address',
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `network_address` varchar(32) NOT NULL DEFAULT '' ,
  `disk_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `free_disk` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `name` varchar(32) NOT NULL DEFAULT '' ,
  `puk` varchar(1000) NOT NULL DEFAULT '' COMMENT 'public key',
  `last_login_time` int(10) unsigned NOT NULL DEFAULT '0' ,
  `login_times` int(10) unsigned NOT NULL DEFAULT '0' ,
  `register_time` int(10) unsigned NOT NULL DEFAULT '0' ,
  PRIMARY KEY (`id`),
  KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH,
  KEY `IDX_BELONG` (`belong`),
  KEY `IDX_NAME` (`name`),
  KEY `IDX_NETWORK_ADDRESS` (`network_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


CREATE TABLE `user_has_file` (
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  PRIMARY KEY (`wallet_address`,`file_hash`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `file_slice_storage` (
  `slice_hash` char(64) NOT NULL DEFAULT '' ,
  `wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'storage pp wallet address',
  `network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'storage pp network address',
  PRIMARY KEY (`slice_hash`,`wallet_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

alter table file add column download int unsigned not null default '0' comment 'file download times' after state;

alter table pp add column free_disk bigint unsigned not null default '0' comment 'free disk size' after disk_size;

insert into user_has_file (wallet_address, file_hash) select owner, hash from file;
alter table file drop column owner;

insert into file_slice_storage (slice_hash, wallet_address, network_address) select slice_hash, wallet_address, network_address from file_slice;
alter table file_slice drop column wallet_address;
alter table file_slice drop column network_address;

rename table download_record to file_slice_download;
alter table file_slice_download add column slice_hash char(64) not null default '' comment 'slice hash' after id;
update file_slice_download as fsd join file_slice as fs on fsd.file_slice_id = fs.id set fsd.slice_hash = fs.slice_hash;
alter table file_slice_download drop column file_slice_id;

alter table transfer_record add column slice_hash char(64) not null default '' comment 'slice hash' after id;
update transfer_record as tr join file_slice as fs on tr.file_slice_id = fs.id set tr.slice_hash = fs.slice_hash;
alter table transfer_record drop column file_slice_id;

CREATE TABLE `file_download` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `file_hash` char(64) NOT NULL DEFAULT '' COMMENT 'file hash',
  `to_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'file downloader wallet address',
  `task_id` char(64) NOT NULL DEFAULT '' COMMENT 'download ID',
  `time` int(11) NOT NULL DEFAULT '0' COMMENT 'download time',
  PRIMARY KEY (`id`),
  KEY `IDX_FILE_HASH` (`file_hash`) USING HASH,
  KEY `IDX_TASK_ID` (`task_id`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create unique index `IDX_SLICE_HASH` on file_slice(slice_hash);

-- 1.4
CREATE TABLE `user_directory` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `wallet_address` char(42) DEFAULT '' COMMENT 'user wallet address',
  `path` varchar(512) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'creation time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_WALLET_ADDRESS_PATH` (`wallet_address`,`path`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `user_directory_map_file` (
  `dir_hash` char(64) NOT NULL DEFAULT '' COMMENT 'directory hash',
  `file_hash` char(64) NOT NULL DEFAULT '' COMMENT 'file hash',
  `owner` char(42) NOT NULL DEFAULT '' COMMENT 'owner wallet address',
  PRIMARY KEY (`dir_hash`,`file_hash`) USING HASH,
  KEY `IDX_WALLET_ADDRESS` (`owner`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

alter table user modify column capacity bigint unsigned not null default '10737418240' comment 'capacity, unit: Bytes';
alter table user modify column used_capacity bigint unsigned not null default '0' comment 'used capacity, unit: BYtes';

-- 1.5
alter table file add column is_cover tinyint unsigned not null default '0' comment 'is cover';
insert into user (is_pp, belong, wallet_address, network_address) values (0, '', '0x0000000000000000000000000000000000000000', '127.0.0.1:9890');
alter table album_has_file add column sort int unsigned not null default '0' comment 'sort';

CREATE TABLE `client_download_record` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `type` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:windows，1:mac',
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

