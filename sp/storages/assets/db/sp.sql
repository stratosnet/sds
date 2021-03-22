-- MySQL dump 10.13  Distrib 5.7.23, for macos10.13 (x86_64)
--
-- Host: localhost    Database: spacebook_sp
-- ------------------------------------------------------
-- Server version	5.7.23

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `spacebook_sp`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `spacebook_sp` /*!40100 DEFAULT CHARACTER SET utf8 */;

USE `spacebook_sp`;

--
-- Table structure for table `album`
--

DROP TABLE IF EXISTS `album`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `album` (
  `album_id` char(16) NOT NULL DEFAULT '' ,
  `name` varchar(64) NOT NULL DEFAULT '' ,
  `introduction` varchar(1000) NOT NULL DEFAULT '' ,
  `cover` char(64) NOT NULL DEFAULT '' ,
  `type` tinyint(3) unsigned NOT NULL DEFAULT '2' ,
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `visit_count` int(10) unsigned NOT NULL DEFAULT '0' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'creation time',
  `state` tinyint(3) unsigned NOT NULL DEFAULT '0' ,
  `is_private` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:private,1:public',
  PRIMARY KEY (`album_id`),
  KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH,
  KEY `IDX_TIME` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `album_has_file`
--

DROP TABLE IF EXISTS `album_has_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `album_has_file` (
  `album_id` char(16) NOT NULL DEFAULT '' ,
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'creation time',
  PRIMARY KEY (`album_id`,`file_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `file`
--

DROP TABLE IF EXISTS `file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(128) NOT NULL DEFAULT '' ,
  `hash` char(64) NOT NULL DEFAULT '' ,
  `size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `slice_num` int(10) unsigned NOT NULL DEFAULT '0' ,
  `state` tinyint(3) unsigned NOT NULL DEFAULT '0' ,
  `download` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'download count',
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'upload time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_HASH` (`hash`) USING HASH,
  KEY `IDX_NAME` (`name`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `file_download`
--

DROP TABLE IF EXISTS `file_download`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `file_slice`
--

DROP TABLE IF EXISTS `file_slice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file_slice` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  `slice_hash` char(64) NOT NULL DEFAULT '' ,
  `slice_size` bigint(20) NOT NULL DEFAULT '0' ,
  `slice_number` int(10) unsigned NOT NULL DEFAULT '1' ,
  `slice_offset_start` bigint(20) NOT NULL DEFAULT '0' ,
  `slice_offset_end` bigint(20) NOT NULL DEFAULT '0' ,
  `status` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:success,1:pending,2:error',
  `task_id` char(64) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'upload time',
  PRIMARY KEY (`id`),
  KEY `IDX_FILE_HASH` (`file_hash`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `file_slice_download`
--

DROP TABLE IF EXISTS `file_slice_download`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
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
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `file_slice_storage`
--

DROP TABLE IF EXISTS `file_slice_storage`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file_slice_storage` (
  `slice_hash` char(64) NOT NULL DEFAULT '' ,
  `wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'storage PP wallet address',
  `network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'storage PP network address',
  PRIMARY KEY (`slice_hash`,`wallet_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pp`
--

DROP TABLE IF EXISTS `pp`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pp` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'Id of pp',
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `network_address` varchar(32) NOT NULL DEFAULT '' ,
  `disk_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `free_disk` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `memory_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `os_and_ver` varchar(128) NOT NULL DEFAULT '' ,
  `cpu_info` varchar(64) NOT NULL DEFAULT '' ,
  `mac_address` varchar(17) NOT NULL DEFAULT '' ,
  `version` int(10) unsigned NOT NULL DEFAULT '0' ,
  `pub_key` varchar(1000) NOT NULL DEFAULT '' ,
  `state` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:offline,1:online',
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `transfer_record`
--

DROP TABLE IF EXISTS `transfer_record`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `transfer_record` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `slice_hash` char(64) NOT NULL DEFAULT '' ,
  `transfer_cer` char(64) NOT NULL DEFAULT '' ,
  `from_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'origin PP wallet address',
  `to_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'target PP wallet address',
  `from_network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'origin PP network address',
  `to_network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'target network address',
  `status` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:success,1:waiting,2:pending,3:error',
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'transfer finish time',
  PRIMARY KEY (`id`),
  KEY `IDX_TRANSFER_CER` (`transfer_cer`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user`
--

DROP TABLE IF EXISTS `user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `is_pp` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:no, 1:yest',
  `belong` char(42) NOT NULL DEFAULT '' COMMENT 'parent wallet address',
  `wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'wallet address',
  `network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'network address',
  `disk_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `capacity` int(10) unsigned NOT NULL DEFAULT '10240' COMMENT 'unit:MB',
  `used_capacity` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'unit:MB',
  `is_upgrade` tinyint(2) unsigned NOT NULL DEFAULT '0' ,
  `be_invited` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT 'is invited',
  `invitation_code` char(8) NOT NULL DEFAULT '' ,
  `free_disk` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `name` varchar(32) NOT NULL DEFAULT '' ,
  `puk` varchar(1000) NOT NULL DEFAULT '' COMMENT 'public key',
  `last_login_time` int(10) unsigned NOT NULL DEFAULT '0' ,
  `login_times` int(10) unsigned NOT NULL DEFAULT '0' ,
  `register_time` int(10) unsigned NOT NULL DEFAULT '0' ,
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_INVITATION_CODE` (`invitation_code`),
  KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH,
  KEY `IDX_BELONG` (`belong`),
  KEY `IDX_NAME` (`name`),
  KEY `IDX_NETWORK_ADDRESS` (`network_address`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_collect_album`
--

DROP TABLE IF EXISTS `user_collect_album`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_collect_album` (
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `album_id` char(16) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'collect time',
  PRIMARY KEY (`wallet_address`,`album_id`),
  KEY `album_id` (`album_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_directory`
--

DROP TABLE IF EXISTS `user_directory`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_directory` (
  `dir_hash` char(64) NOT NULL DEFAULT '' COMMENT 'path + wd',
  `wallet_address` char(42) DEFAULT '' ,
  `path` varchar(512) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'creation time',
  PRIMARY KEY (`dir_hash`),
  UNIQUE KEY `IDX_WALLET_ADDRESS_PATH` (`wallet_address`,`path`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_directory_map_file`
--

DROP TABLE IF EXISTS `user_directory_map_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_directory_map_file` (
  `dir_hash` char(64) NOT NULL DEFAULT '' ,
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  PRIMARY KEY (`dir_hash`,`file_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_has_file`
--

DROP TABLE IF EXISTS `user_has_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_has_file` (
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `file_hash` char(64) NOT NULL DEFAULT '' ,
  PRIMARY KEY (`wallet_address`,`file_hash`) USING HASH,
  KEY `FK_FILE_HASH_HASH` (`file_hash`),
  CONSTRAINT `FK_FILE_HASH_HASH` FOREIGN KEY (`file_hash`) REFERENCES `file` (`hash`),
  CONSTRAINT `FK_WALLET_ADDRESS` FOREIGN KEY (`wallet_address`) REFERENCES `user` (`wallet_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_invite`
--

DROP TABLE IF EXISTS `user_invite`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_invite` (
  `invitation_code` char(8) NOT NULL DEFAULT '' ,
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `times` tinyint(3) unsigned NOT NULL DEFAULT '0' ,
  PRIMARY KEY (`invitation_code`),
  UNIQUE KEY `IDX_INVITATION_CODE` (`invitation_code`) USING HASH,
  UNIQUE KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_invite_record`
--

DROP TABLE IF EXISTS `user_invite_record`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_invite_record` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `invitation_code` char(8) NOT NULL DEFAULT '' ,
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `reward` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'reward capacity',
  `time` int(10) unsigned NOT NULL DEFAULT '0' ,
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_WALLET_ADDRESS` (`wallet_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_share`
--

DROP TABLE IF EXISTS `user_share`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_share` (
  `share_id` char(16) NOT NULL DEFAULT '' ,
  `rand_code` char(6) NOT NULL DEFAULT '' COMMENT 'random code',
  `open_type` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:public,1:private',
  `deadline` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '0:forever',
  `share_type` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:file,1:directory',
  `password` char(4) NOT NULL DEFAULT '' ,
  `hash` char(64) NOT NULL DEFAULT '' COMMENT 'content hash， share_type=0:file hash，share_type=1:directory hash',
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'share creation time',
  PRIMARY KEY (`share_id`),
  KEY `IDX_RAND_CODE` (`rand_code`) USING BTREE,
  KEY `IDX_FILE_HASH` (`hash`) USING HASH,
  KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2018-11-17 23:35:50
