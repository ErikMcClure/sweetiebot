DELIMITER //

SET NAMES utf8mb4//
SET GLOBAL log_bin_trust_function_creators = 1//

-- Dumping database structure for sweetiebot
CREATE DATABASE IF NOT EXISTS `sweetiebot` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci//
ALTER DATABASE `sweetiebot` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci //
USE `sweetiebot`//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.timezones
CREATE TABLE IF NOT EXISTS `timezones` (
  `Location` varchar(40) NOT NULL,
  `Offset` int(11) NOT NULL,
  `DST` int(11) NOT NULL,
  PRIMARY KEY (`Location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.users
CREATE TABLE IF NOT EXISTS `users` (
  `ID` bigint(20) unsigned NOT NULL,
  `Username` varchar(128) NOT NULL DEFAULT '',
  `Discriminator` int(10) unsigned NOT NULL DEFAULT 0,
  `LastSeen` datetime NOT NULL,
  `LastNameChange` datetime NOT NULL,
  `Location` varchar(40) DEFAULT NULL,
  `DefaultServer` bigint(20) unsigned DEFAULT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_USERNAME` (`Username`),
  KEY `FK_Location_timezone` (`Location`),
  CONSTRAINT `FK_Location_timezone` FOREIGN KEY (`Location`) REFERENCES `timezones` (`Location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

CREATE PROCEDURE `AddChat`(
	IN `_id` BIGINT,
	IN `_author` BIGINT,
	IN `_username` VARCHAR(128),
	IN `_message` VARCHAR(2000),
	IN `_channel` BIGINT,
	IN `_guild` BIGINT

)
MODIFIES SQL DATA
BEGIN

INSERT INTO users (ID, Username, LastSeen, LastNameChange) 
VALUES (_author, _username, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE LastSeen=UTC_TIMESTAMP();

INSERT IGNORE INTO aliases (`User`, Alias, Duration, `Timestamp`)
VALUES (_id, _username, 0, UTC_TIMESTAMP());

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Guild)
VALUES (_id, _author, _message, UTC_TIMESTAMP(), _channel, _guild)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = UTC_TIMESTAMP();

END//


CREATE FUNCTION `AddItem`(`_content` VARCHAR(500)) RETURNS bigint(20)
    MODIFIES SQL DATA
BEGIN
SET @id = (SELECT ID FROM items WHERE Content = _content);

IF @id IS NULL THEN
	INSERT INTO items (Content) VALUES (_content);
	RETURN LAST_INSERT_ID();
END IF;

RETURN @id;
END//

CREATE PROCEDURE `AddMember`(IN `_id` BIGINT, IN `_guild` BIGINT, IN `_firstseen` DATETIME, IN `_nickname` VARCHAR(128))
    MODIFIES SQL DATA
INSERT INTO members (ID, Guild, FirstSeen, Nickname)
VALUES (_id, _guild, _firstseen, _nickname)
ON DUPLICATE KEY UPDATE
FirstSeen=GetMinDate(_firstseen,FirstSeen), Nickname=_nickname//

CREATE PROCEDURE `AddUser`(
	IN `_id` BIGINT,
	IN `_username` VARCHAR(128),
	IN `_discriminator` INT,
	IN `_isonline` BIT
)
    MODIFIES SQL DATA
BEGIN

DECLARE oldname VARCHAR(128) DEFAULT '';
SELECT Username INTO oldname
FROM users
WHERE ID = _id FOR UPDATE;

INSERT INTO users (ID, Username, Discriminator, LastSeen, LastNameChange) 
VALUES (_id, _username, _discriminator, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE 
Username=IF(_username = '', Username, _username),
Discriminator=IF(_discriminator = 0, Discriminator, _discriminator),
LastSeen=IF(_isonline > 0, UTC_TIMESTAMP(), LastSeen),
LastNameChange=IF(_username = '', LastNameChange, IF(_username != `Username`, UTC_TIMESTAMP(), LastNameChange));

IF _username != '' THEN
	IF oldname != '' THEN
		INSERT INTO aliases (`User`, Alias, Duration, `Timestamp`)
		VALUES (_id, oldname, 0, UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE `Duration` = `Duration` + (UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(`Timestamp`)), `Timestamp` = UTC_TIMESTAMP(); 
	END IF;
	
	IF _username != oldname THEN
		INSERT INTO aliases (`User`, Alias, Duration, `Timestamp`)
		VALUES (_id, _username, 0, UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE `Timestamp` = UTC_TIMESTAMP(); 
	END IF;
END IF;

END//

-- Dumping structure for table sweetiebot.aliases
CREATE TABLE IF NOT EXISTS `aliases` (
  `User` bigint(20) unsigned NOT NULL,
  `Alias` varchar(128) NOT NULL,
  `Timestamp` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `Duration` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`User`,`Alias`),
  KEY `ALIASES_USERS` (`User`),
  KEY `ALIASES_ALIAS` (`Alias`),
  CONSTRAINT `ALIASES_USERS` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.chatlog
CREATE TABLE IF NOT EXISTS `chatlog` (
  `ID` bigint(20) unsigned NOT NULL,
  `Author` bigint(20) unsigned NOT NULL,
  `Message` varchar(2000) NOT NULL,
  `Timestamp` datetime NOT NULL,
  `Channel` bigint(20) unsigned NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `INDEX_CHANNEL` (`Channel`),
  KEY `CHATLOG_USERS` (`Author`),
  CONSTRAINT `CHATLOG_USERS` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='A log of all the messages from all the chatrooms.'//


CREATE EVENT `CleanAliases`
	ON SCHEDULE
		EVERY 1 DAY STARTS '2018-01-23 16:29:25'
	ON COMPLETION PRESERVE
	ENABLE
	COMMENT ''
	DO BEGIN
  
DELETE FROM chatlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
DELETE FROM debuglog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 8 DAY);
DELETE FROM users WHERE `ID` NOT IN (SELECT DISTINCT ID FROM members) AND `ID` NOT IN (SELECT DISTINCT Author FROM chatlog);

Block1: BEGIN
DECLARE done INT DEFAULT 0;
DECLARE uid BIGINT UNSIGNED;
DECLARE aid VARCHAR(128);
DECLARE c_1 CURSOR FOR SELECT User FROM aliases GROUP BY User HAVING COUNT(Alias) > 10;
DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

OPEN c_1;
REPEAT
FETCH c_1 INTO uid;

Block2: BEGIN
DECLARE done2 INT DEFAULT 0;
DECLARE c_2 CURSOR FOR SELECT A.Alias FROM aliases A INNER JOIN users U ON A.User = U.ID WHERE A.User = uid AND A.Alias != U.Username ORDER BY A.Duration DESC LIMIT 9999 OFFSET 10;
DECLARE CONTINUE HANDLER FOR NOT FOUND SET done2 = 1;
OPEN c_2;
REPEAT
FETCH c_2 INTO aid;

DELETE FROM aliases WHERE User=uid AND Alias=aid;

UNTIL done2 END REPEAT;
CLOSE c_2;
END Block2;

UNTIL done END REPEAT;
CLOSE c_1;
END Block1;

END//

-- Dumping structure for table sweetiebot.debuglog
CREATE TABLE IF NOT EXISTS `debuglog` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Type` tinyint(3) unsigned NOT NULL,
  `User` bigint(20) unsigned DEFAULT NULL,
  `Message` varchar(4096) NOT NULL,
  `Timestamp` datetime NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `debuglog_Users` (`User`),
  CONSTRAINT `debuglog_Users` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

CREATE FUNCTION `GetMinDate`(`date1` DATETIME, `date2` DATETIME) RETURNS datetime
    NO SQL
    DETERMINISTIC
BEGIN
IF date1 = '0000-00-00 00:00:00' THEN
	RETURN date2;
END IF;
IF date2 = '0000-00-00 00:00:00' THEN
	RETURN date1;
END IF;
IF date1 < date2 THEN
	RETURN date1;
END IF;
RETURN date2;
END//

-- Dumping structure for table sweetiebot.items
CREATE TABLE IF NOT EXISTS `items` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Content` varchar(500) NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `CONTENT_INDEX` (`Content`(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.tags
CREATE TABLE IF NOT EXISTS `tags` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Name` varchar(50) NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `UNIQUE_NAME` (`Name`,`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.itemtags
CREATE TABLE IF NOT EXISTS `itemtags` (
  `Item` bigint(20) unsigned NOT NULL,
  `Tag` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`Item`,`Tag`),
  KEY `FK_itemtags_tags` (`Tag`),
  CONSTRAINT `FK_itemtags_items` FOREIGN KEY (`Item`) REFERENCES `items` (`ID`),
  CONSTRAINT `FK_itemtags_tags` FOREIGN KEY (`Tag`) REFERENCES `tags` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.members
CREATE TABLE IF NOT EXISTS `members` (
  `ID` bigint(20) unsigned NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  `FirstSeen` datetime NOT NULL,
  `Nickname` varchar(128) NOT NULL DEFAULT '',
  `FirstMessage` datetime DEFAULT NULL,
  PRIMARY KEY (`ID`,`Guild`),
  KEY `INDEX_NICKNAME` (`Nickname`),
  KEY `INDEX_GUILD_FIRSTSEEN` (`Guild`,`FirstSeen`),
  CONSTRAINT `FK_members_users` FOREIGN KEY (`ID`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

CREATE PROCEDURE `RemoveGuild`(
	IN `_guild` BIGINT UNSIGNED
)
    MODIFIES SQL DATA
BEGIN

DELETE FROM `members` WHERE Guild = _guild;
DELETE FROM `schedule` WHERE Guild = _guild;
DELETE FROM `chatlog` WHERE Guild = _guild;
DELETE FROM `debuglog` WHERE Guild = _guild;
DELETE FROM `tags` WHERE Guild = _guild;

END//

CREATE PROCEDURE `RemoveSchedule`(IN `_id` BIGINT)
    MODIFIES SQL DATA
BEGIN
DELETE FROM `schedule` WHERE ID = _id AND Date > UTC_TIMESTAMP();
DELETE FROM `schedule` WHERE ID = _id AND `Repeat` IS NULL AND `RepeatInterval` IS NULL;
CASE (SELECT `RepeatInterval` FROM `schedule` WHERE ID = _id)
	WHEN 1 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` SECOND) WHERE ID = _id;
	WHEN 2 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` MINUTE) WHERE ID = _id;
	WHEN 3 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` HOUR) WHERE ID = _id;
	WHEN 4 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` DAY) WHERE ID = _id;
	WHEN 5 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` WEEK) WHERE ID = _id;
	WHEN 6 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` MONTH) WHERE ID = _id;
	WHEN 7 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` QUARTER) WHERE ID = _id;
	WHEN 8 THEN UPDATE `schedule` SET Date = DATE_ADD(Date, INTERVAL `Repeat` YEAR) WHERE ID = _id;
	ELSE BEGIN END;
END CASE;
END//

-- Dumping structure for table sweetiebot.schedule
CREATE TABLE IF NOT EXISTS `schedule` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Guild` bigint(20) unsigned NOT NULL,
  `Date` datetime NOT NULL,
  `RepeatInterval` tinyint(3) unsigned DEFAULT NULL,
  `Repeat` int(11) DEFAULT NULL,
  `Type` tinyint(3) unsigned NOT NULL,
  `Data` text NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_GUILD_DATE_TYPE` (`Date`,`Guild`,`Type`),
  KEY `INDEX_GUILD` (`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.transcripts
CREATE TABLE IF NOT EXISTS `transcripts` (
  `Season` int(10) unsigned NOT NULL,
  `Episode` int(10) unsigned NOT NULL,
  `Line` int(10) unsigned NOT NULL,
  `Speaker` varchar(128) NOT NULL,
  `Text` varchar(2000) NOT NULL,
  PRIMARY KEY (`Season`,`Episode`,`Line`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Dumping structure for trigger sweetiebot.itemtags_after_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `itemtags_after_delete` AFTER DELETE ON `itemtags` FOR EACH ROW BEGIN

IF (SELECT COUNT(*) FROM itemtags WHERE Item = OLD.Item) = 0 THEN
DELETE FROM items WHERE ID = OLD.Item;
END IF;

END//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.tags_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `tags_before_delete` BEFORE DELETE ON `tags` FOR EACH ROW DELETE FROM itemtags WHERE Tag = OLD.ID//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.users_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members or chatlog tables
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;

END//
SET SQL_MODE=@OLDTMP_SQL_MODE//
