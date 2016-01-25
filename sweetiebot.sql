-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               10.1.10-MariaDB - mariadb.org binary distribution
-- Server OS:                    Win64
-- HeidiSQL Version:             9.1.0.4867
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Dumping database structure for sweetiebot
CREATE DATABASE IF NOT EXISTS `sweetiebot` /*!40100 DEFAULT CHARACTER SET utf8 */;
USE `sweetiebot`;


-- Dumping structure for procedure sweetiebot.AddChat
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddChat`(IN `id` BIGINT, IN `author` BIGINT, IN `message` VARCHAR(2000), IN `channel` BIGINT, IN `everyone` BIT)
BEGIN

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Everyone)
VALUES (id, author, message, Now(6), channel, everyone);

END//
DELIMITER ;


-- Dumping structure for procedure sweetiebot.AddUser
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddUser`(IN `id` BIGINT, IN `email` VARCHAR(512), IN `username` VARCHAR(512), IN `avatar` VARCHAR(512), IN `verified` BIT)
    DETERMINISTIC
BEGIN

INSERT INTO users (ID, Email, Username, Avatar, Verified, FirstSeen, LastSeen)
VALUES (id, email, username, avatar, verified, Now(6), Now(6))
ON DUPLICATE KEY UPDATE
Username=username, Avatar=avatar, Verified=verified, LastSeen=Now(6);

END//
DELIMITER ;


-- Dumping structure for table sweetiebot.chatlog
CREATE TABLE IF NOT EXISTS `chatlog` (
  `ID` bigint(20) unsigned NOT NULL,
  `Author` bigint(20) unsigned NOT NULL,
  `Message` varchar(2000) NOT NULL,
  `Timestamp` datetime NOT NULL,
  `Channel` bigint(20) unsigned NOT NULL,
  `Everyone` bit(1) NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `INDEX_CHANNEL` (`Channel`),
  KEY `CHATLOG_USERS` (`Author`),
  CONSTRAINT `CHATLOG_USERS` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='A log of all the messages from all the chatrooms.';

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.debuglog
CREATE TABLE IF NOT EXISTS `debuglog` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Message` varchar(2048) NOT NULL,
  `Timestamp` datetime NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.pings
CREATE TABLE IF NOT EXISTS `pings` (
  `Message` bigint(20) unsigned NOT NULL,
  `User` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`Message`,`User`),
  KEY `PINGS_USERS` (`User`),
  CONSTRAINT `PINGS_CHATLOG` FOREIGN KEY (`Message`) REFERENCES `chatlog` (`ID`),
  CONSTRAINT `PINGS_USERS` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.users
CREATE TABLE IF NOT EXISTS `users` (
  `ID` bigint(20) unsigned NOT NULL,
  `Email` varchar(512) NOT NULL,
  `Username` varchar(512) NOT NULL,
  `Avatar` varchar(512) NOT NULL,
  `Verified` bit(1) NOT NULL,
  `FirstSeen` datetime NOT NULL,
  `LastSeen` datetime NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_USERNAME` (`Username`(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
