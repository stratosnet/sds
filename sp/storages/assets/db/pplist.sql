-- MySQL dump 10.13  Distrib 5.7.23, for linux-glibc2.12 (x86_64)
--
-- Host: localhost    Database: sds_sp
-- ------------------------------------------------------
-- Server version	5.7.23-log

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
SET @MYSQLDUMP_TEMP_LOG_BIN = @@SESSION.SQL_LOG_BIN;
SET @@SESSION.SQL_LOG_BIN= 0;

--
-- GTID state at the beginning of the backup 
--


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
  `request_network_address` varchar(32) NOT NULL DEFAULT '' ,
  `disk_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `memory_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `os_and_ver` varchar(128) NOT NULL DEFAULT '' ,
  `cpu_info` varchar(64) NOT NULL DEFAULT '' ,
  `mac_address` varchar(17) NOT NULL DEFAULT '' ,
  `version` int(10) unsigned NOT NULL DEFAULT '0' ,
  `pub_key` varchar(1000) NOT NULL DEFAULT '' ,
  `state` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `active` boolean NOT NULL DEFAULT false,
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=64 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pp`
--

LOCK TABLES `pp` WRITE;
/*!40000 ALTER TABLE `pp` DISABLE KEYS */;
INSERT INTO `pp` VALUES (1,'0x216eBD182E3b00848d81Ce532594068D98545774','207.180.201.64:9890','',11899928051712,270370820096,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1	','ac:1f:6b:46:5a:c1',1,'048eea001cd38da62cbc8a7a5954ec07b3a8e767d377217f51b8dd68ef7f8a9dde861f0c31b243fa8fc8e1a2a96a66eea4ad4cc0419b2ded6c49f88e7e12332fba',0),(5,'0x718BBa283b7F8EF9b2BC14F661B7A57F0CD66341','173.212.237.87:9890','',11899928051712,270370820096,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1	','ac:1f:6b:46:5a:c1',1,'04ea3b65812eb23f7d793bde6a29ff1e3bdff94e66bf7e23ae1aa3fd9573b7d8ef77030fdca6a1a12837f42d17ef2a1ec65c9c6977f870a154461d121449e189d7',0),(8,'0xfC0E51FDB1F5008Be2Fec74d66f24F9383856A91','173.249.29.237:9890','',11899928051712,270371086336,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1	','ac:1f:6b:46:5a:55',1,'04d9116ae940908c28fcb0640b38f9a7d14214731cc03b7815ff08ff807632a8915e506b2cd6c836f58c74e6467ad48be71a9f0f1e737a3866411c482362d51a9a',0),(9,'0xFC230ded24a9aeEc0503250373d0FEe84DD1516A','173.212.237.73:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1	','ac:1f:6b:02:28:1b',1,'0446f070113ebb4b81ec88c9dcfd893adf8adcac3b87cf62c68bafce34632f8601d5a57b745fab5aab3e77197a73e20e90bd61f9b0b6711fcbb005f6b9f94adcc9',0),(10,'0x3E4A3B5EE90A4E22f6B11296a3434a56c451e587','107.167.5.34:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(11,'0x726b2d0d37147EAF8FF48D163851D9CBa32D360c','170.178.182.2:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(12,'0xFf73818BD447aD3B44a43f953FFEC9A698F08927','45.58.137.106:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(13,'0xc02a9f6387Dc1e96DC1eCEf3b16e7eE3cbB6623B','45.58.135.178:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(14,'0xb2cA2AA8Be389125cF9DfE358e1521173CeDf4aC','107.167.5.50:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(15,'0x6f754446490e4D03f0576b8100352CbEdaDDc89F','107.167.20.226:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(17,'0x12E239fE4f9F9bAFa5D8A3a22aDba809E0Ae0439','23.234.11.204:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',0),(18,'0xA72c3a84633f1171E0c31950856d3B3beC8e23af','23.234.11.195:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(19,'0xFd3a7DA1fa15845053c95C5F37a17C24c61b064F','23.234.11.150:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(20,'0x78c781974B95982dD6B5A27B4CDFf3A7A9e6Dbc3','23.234.11.128:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(21,'0xfb6a8F8FD18d2E0Ebd73f52d6ab8503B835246FE','23.234.11.183:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(22,'0x45e4F617B763d855b9Afb68aeb641EB41BdBbFe5','23.234.11.137:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(23,'0x2d5e8B7B5668DEA1034A65251b99DCdb68f670b2','23.234.11.179:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(24,'0xA892C6545661911CAeE4d2cebc3fDaA2cC0eF0e4','23.234.11.204:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(25,'0x266fAD2CDAaBF4948c7e02c155DfB97EbdF444dc','23.234.11.156:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(26,'0xB023c851d4ADFa1a44f02bD683e48B8894f7246D','23.234.11.131:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(27,'0x5b56b7dd4433C8C17e43021026336ac5c6Cd07bb','23.234.11.190:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(28,'0x4DD7C82006B9F1D6722f4692a9693E37177931a8','23.234.11.151:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(29,'0xDE9Fa06478A8342FCfA782291aE36377125920d7','23.234.11.175:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(30,'0x27bDF4778506e87afAD7e848c47Ef13F1e224c86','23.234.11.127:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(31,'0x52AFcf55E0278E5b3F332abA87aD6d777e2f8193','23.234.11.167:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(32,'0x99B872dBaa8d80D00269F6D08D55E0794FbC13ff','23.234.11.169:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(33,'0x8EaD4ebB1ddA702c1DED5d51Bceb137bA28Bb3A3','23.234.11.152:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(34,'0x3aF0a41cC37373799bff4fc435E5818cC72d7C4f','23.234.11.186:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(35,'0xE2EA5Dac078DA634F56528c4587bA0C6B95e2fbD','23.234.11.201:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(36,'0x540E9186560561dfABbc1aF421c8046c53A79B59','23.234.11.184:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(37,'0xD7029EDF55d162Ea88984889c94E156eD2cf7b06','23.234.11.149:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(38,'0xfC09D165aB3c941B3054F2B80513a068442886c8','23.234.11.187:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(39,'0x6F1A752ba3F65a831991a0E58E5a4aEb7Bb3C63b','23.234.11.148:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(40,'0x30024E8eF1AB086802AFB6eD0942E7FdFA2eED28','23.234.11.177:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(41,'0xD02e634897BB4c2B2CcB5F11122F310aB83E5e94','23.234.11.165:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(42,'0xB8678298cfae0f5B154046095e9Db0342010df4C','23.234.11.188:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(43,'0xd62e14B0e135f9478538CCE8ce199d3dad78E36F','23.234.11.173:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(44,'0xC7b991Bd444d806B68920eBa250e05382a4E5E8D','23.234.11.129:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(45,'0x562F2790a325E8e005b0C1B9258d58122d6f50Ce','23.234.11.199:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(46,'0x1CE0eebecFD432035D8713aDa0be8d072142fB10','23.234.11.191:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(47,'0x8D8Ad6a27227E9c433B865d2BfAA6427e71c1000','23.234.11.189:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(48,'0xcbb1c47ac0B1BF45D7572c28c698f08C38d7c1CE','23.234.11.174:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(49,'0xae5DAbc59033Adfdf16B614A7836cdfAd45bF4e6','23.234.11.166:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(50,'0x80E08bF8C36851d9e0782897dF5c37c5487Ea86e','23.234.11.172:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(51,'0x6aEa8A9d300ba36a017D7f20a621c2C605f2d35B','23.234.11.62:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(52,'0x9d9D14c6B1587c099a1269b148BD972bF1CC4a76','23.234.11.216:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(53,'0xE7c7c9754Ef1b6207108919b9bB55bd2b3262793','23.234.11.217:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(54,'0x136Bc1F1b0B664b98795bfd4370496b35B9F17Af','23.234.11.202:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(55,'0x5668EA385F6f4234E4AC9328338bf9D27c6802cB','23.234.11.235:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(56,'0x27a574Ae65650183270a55C00D4E64E797153Ba6','23.234.11.164:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(57,'0x4bAC836854433Ebfcf5Edf4fc54975AE1b221d4b','23.234.11.130:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(58,'0x2c9508347672E3cFFDfFBDaB8d2CB3B0f8CDd2D7','23.234.11.132:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(59,'0xD0Ea86b06139B9cE000Fc40F4f20b8Fc74F2ca4B','23.234.11.170:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(60,'0x4c594793427ECD4B0f3581438021DAe283113fb8','23.234.11.182:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(61,'0xC05c8114f8B516e4B6c676Aa7186357ED75859dB','23.234.11.197:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(62,'0xBBf5f2A9A2BAc95F28708ecA915a4a80A6998A3d','23.234.11.171:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1),(63,'0x999aE343980D80d2A59AffC19d1801eFd489c1Da','23.234.11.168:9890','',11899928051712,270371115008,'centos rhel 7.5.1804','Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz 1','ac:1f:6b:02:28:1b',1,'',1);
/*!40000 ALTER TABLE `pp` ENABLE KEYS */;
UNLOCK TABLES;
SET @@SESSION.SQL_LOG_BIN = @MYSQLDUMP_TEMP_LOG_BIN;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2018-10-29  3:12:44
