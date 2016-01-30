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
CREATE DATABASE IF NOT EXISTS `sweetiebot` /*!40100 DEFAULT CHARACTER SET utf8mb4 */;
USE `sweetiebot`;


-- Dumping structure for procedure sweetiebot.AddChat
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddChat`(IN `_id` BIGINT, IN `_author` BIGINT, IN `_message` VARCHAR(2000), IN `_channel` BIGINT, IN `_everyone` BIT)
    DETERMINISTIC
BEGIN

CALL SawUser(_author);

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Everyone)
VALUES (_id, _author, _message, Now(6), _channel, _everyone)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = Now(6), Everyone=_everyone;

END//
DELIMITER ;


-- Dumping structure for procedure sweetiebot.AddUser
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddUser`(IN `_id` BIGINT, IN `_email` VARCHAR(512), IN `_username` VARCHAR(512), IN `_avatar` VARCHAR(512), IN `_verified` BIT)
    DETERMINISTIC
INSERT INTO users (ID, Email, Username, Avatar, Verified, FirstSeen, LastSeen, LastNameChange)
VALUES (_id, _email, _username, _avatar, _verified, Now(6), Now(6), Now(6))
ON DUPLICATE KEY UPDATE
Username=_username, Avatar=_avatar, Verified=_verified, LastSeen=Now(6)//
DELIMITER ;


-- Dumping structure for table sweetiebot.aliases
CREATE TABLE IF NOT EXISTS `aliases` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `User` bigint(20) unsigned NOT NULL,
  `Alias` varchar(128) NOT NULL,
  `Duration` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `ALIASES_ALIAS` (`Alias`),
  KEY `ALIASES_USERS` (`User`),
  CONSTRAINT `ALIASES_USERS` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.chatlog
CREATE TABLE IF NOT EXISTS `chatlog` (
  `ID` bigint(20) unsigned NOT NULL DEFAULT '0',
  `Author` bigint(20) unsigned NOT NULL DEFAULT '0',
  `Message` varchar(2000) NOT NULL DEFAULT '',
  `Timestamp` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  `Channel` bigint(20) unsigned NOT NULL DEFAULT '0',
  `Everyone` bit(1) NOT NULL DEFAULT b'0',
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `INDEX_CHANNEL` (`Channel`),
  KEY `CHATLOG_USERS` (`Author`),
  CONSTRAINT `CHATLOG_USERS` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='A log of all the messages from all the chatrooms.';

-- Data exporting was unselected.


-- Dumping structure for event sweetiebot.CleanChatlog
DELIMITER //
CREATE DEFINER=`root`@`localhost` EVENT `CleanChatlog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:04:34' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM chatlog WHERE Timestamp < DATE_SUB(NOW(6), INTERVAL 7 DAY);
END//
DELIMITER ;


-- Dumping structure for event sweetiebot.CleanDebugLog
DELIMITER //
CREATE DEFINER=`root`@`localhost` EVENT `CleanDebugLog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:30:36' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM debuglog WHERE Timestamp < DATE_SUB(NOW(6), INTERVAL 8 DAY);
END//
DELIMITER ;


-- Dumping structure for table sweetiebot.debuglog
CREATE TABLE IF NOT EXISTS `debuglog` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Message` varchar(2048) NOT NULL,
  `Timestamp` datetime NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.editlog
CREATE TABLE IF NOT EXISTS `editlog` (
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
  CONSTRAINT `EDITLOG_CHATLOG` FOREIGN KEY (`ID`) REFERENCES `chatlog` (`ID`),
  CONSTRAINT `editlog_ibfk_1` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=COMPACT COMMENT='A log of all the messages from all the chatrooms.';

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.pings
CREATE TABLE IF NOT EXISTS `pings` (
  `Message` bigint(20) unsigned NOT NULL,
  `User` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`Message`,`User`),
  KEY `PINGS_USERS` (`User`),
  CONSTRAINT `PINGS_CHATLOG` FOREIGN KEY (`Message`) REFERENCES `chatlog` (`ID`),
  CONSTRAINT `PINGS_USERS` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for procedure sweetiebot.SawUser
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `SawUser`(IN `_id` BIGINT)
BEGIN

INSERT INTO users (ID, Email, Username, Avatar, Verified, FirstSeen, LastSeen)
VALUES (_id, '', '', '', 0, Now(6), Now(6))
ON DUPLICATE KEY UPDATE LastSeen=Now(6);

END//
DELIMITER ;


-- Dumping structure for procedure sweetiebot.UpdateUserJoinTime
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `UpdateUserJoinTime`(IN `_user` BIGINT, IN `_joinedat` DATETIME)
BEGIN

UPDATE users SET FirstSeen = _joinedat WHERE ID = _user AND _joinedat < FirstSeen;

END//
DELIMITER ;


-- Dumping structure for table sweetiebot.users
CREATE TABLE IF NOT EXISTS `users` (
  `ID` bigint(20) unsigned NOT NULL DEFAULT '0',
  `Email` varchar(512) NOT NULL DEFAULT '',
  `Username` varchar(128) NOT NULL DEFAULT '',
  `Avatar` varchar(512) NOT NULL DEFAULT '',
  `Verified` bit(1) NOT NULL DEFAULT b'0',
  `FirstSeen` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  `LastSeen` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  `LastNameChange` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`ID`),
  KEY `INDEX_USERNAME` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for trigger sweetiebot.chatlog_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `chatlog_before_delete` BEFORE DELETE ON `chatlog` FOR EACH ROW BEGIN
DELETE FROM pings WHERE Message = OLD.ID;
DELETE FROM editlog WHERE ID = OLD.ID;
END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.chatlog_before_update
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW BEGIN

INSERT INTO editlog (ID, Author, Message, Timestamp, Channel, Everyone)
VALUES (OLD.ID, OLD.Author, OLD.Message, OLD.Timestamp, OLD.Channel, OLD.Everyone)
ON DUPLICATE KEY UPDATE ID = OLD.ID;

END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.users_before_update
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `users_before_update` BEFORE UPDATE ON `users` FOR EACH ROW BEGIN

IF NEW.Username = '' THEN
	SET NEW.Username = OLD.Username;
END IF;

IF NEW.Avatar = '' THEN
	SET NEW.Avatar = OLD.Avatar;
END IF;

IF NEW.Username != OLD.Username THEN
	SET NEW.LastNameChange = NOW(6);
	SET @diff = UNIX_TIMESTAMP(NEW.LastNameChange) - UNIX_TIMESTAMP(OLD.LastNameChange);
	INSERT INTO aliases (User, Alias, Duration)
	VALUES (OLD.ID, OLD.Username, @diff)
	ON DUPLICATE KEY UPDATE Duration = Duration + @diff;
END IF;

END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
