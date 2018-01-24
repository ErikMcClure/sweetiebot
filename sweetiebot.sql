DELIMITER //

SET NAMES utf8mb4//

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
  `Avatar` varchar(512) NOT NULL DEFAULT '',
  `LastSeen` datetime NOT NULL,
  `LastNameChange` datetime NOT NULL,
  `Location` varchar(40) DEFAULT NULL,
  `DefaultServer` bigint(20) unsigned DEFAULT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_USERNAME` (`Username`),
  KEY `FK_Location_timezone` (`Location`),
  CONSTRAINT `FK_Location_timezone` FOREIGN KEY (`Location`) REFERENCES `timezones` (`Location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

CREATE PROCEDURE `AddChat`(IN `_id` BIGINT, IN `_author` BIGINT, IN `_message` VARCHAR(2000), IN `_channel` BIGINT, IN `_everyone` BIT, IN `_guild` BIGINT)
    MODIFIES SQL DATA
BEGIN

INSERT INTO users (ID, Username, Avatar, LastSeen, LastNameChange) 
VALUES (_author, '', '', UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE LastSeen=UTC_TIMESTAMP();

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Everyone, Guild)
VALUES (_id, _author, _message, UTC_TIMESTAMP(), _channel, _everyone, _guild)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = UTC_TIMESTAMP(), Everyone=_everyone;

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

CREATE FUNCTION `AddMarkov`(`_prev` BIGINT, `_prev2` BIGINT, `_speaker` VARCHAR(64), `_phrase` VARCHAR(64)) RETURNS bigint(20)
    MODIFIES SQL DATA
    DETERMINISTIC
BEGIN

INSERT INTO markov_transcripts_speaker (Speaker)
VALUES (_speaker)
ON DUPLICATE KEY UPDATE ID = ID;

SET @speakerid = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = _speaker);

INSERT INTO markov_transcripts (SpeakerID, Phrase)
VALUES (@speakerid, _phrase)
ON DUPLICATE KEY UPDATE Phrase = _phrase;

/*LAST_UPDATE_ID() doesn't work here because of the duplicate key possibility */
SET @ret = (SELECT ID FROM markov_transcripts WHERE SpeakerID = @speakerid AND Phrase = _phrase);

INSERT INTO markov_transcripts_map (Prev, Prev2, Next)
VALUES (_prev, _prev2, @ret)
ON DUPLICATE KEY UPDATE Count = Count + 1;

RETURN @ret;
END//

CREATE PROCEDURE `AddMember`(IN `_id` BIGINT, IN `_guild` BIGINT, IN `_firstseen` DATETIME, IN `_nickname` VARCHAR(128))
    MODIFIES SQL DATA
INSERT INTO members (ID, Guild, FirstSeen, Nickname)
VALUES (_id, _guild, _firstseen, _nickname)
ON DUPLICATE KEY UPDATE
FirstSeen=GetMinDate(_firstseen,FirstSeen), Nickname=_nickname//

CREATE PROCEDURE `AddUser`(
	IN `_id` BIGINT,
	IN `_username` VARCHAR(512),
	IN `_discriminator` INT,
	IN `_avatar` VARCHAR(512),
	IN `_isonline` BIT
)
    MODIFIES SQL DATA
BEGIN

DECLARE oldname VARCHAR(128) DEFAULT '';
SELECT Username INTO oldname
FROM users
WHERE ID = _id FOR UPDATE;

INSERT INTO users (ID, Username, Discriminator, Avatar, LastSeen, LastNameChange) 
VALUES (_id, _username, _discriminator, _avatar, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE 
Username=IF(_username = '', Username, _username),
Discriminator=IF(_discriminator = 0, Discriminator, _discriminator),
Avatar=IF(_avatar = '', Avatar, _avatar),
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
  `Timestamp` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
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
  `Everyone` bit(1) NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `INDEX_CHANNEL` (`Channel`),
  KEY `CHATLOG_USERS` (`Author`),
  CONSTRAINT `CHATLOG_USERS` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='A log of all the messages from all the chatrooms.'//

CREATE EVENT `CleanChatlog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:04:34' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM chatlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
DELETE FROM editlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
END//

CREATE EVENT `CleanDebugLog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:30:36' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM debuglog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 8 DAY);
END//

CREATE EVENT `CleanUsers` ON SCHEDULE EVERY 1 DAY STARTS '2018-01-22 15:50:17' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM users WHERE `ID` NOT IN (SELECT DISTINCT ID FROM members);
END//

CREATE EVENT `CleanAliases`
	ON SCHEDULE
		EVERY 1 DAY STARTS '2018-01-23 16:29:25'
	ON COMPLETION PRESERVE
	ENABLE
	COMMENT ''
	DO BEGIN

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

-- Dumping structure for table sweetiebot.editlog
CREATE TABLE IF NOT EXISTS `editlog` (
  `ID` BIGINT(20) UNSIGNED NOT NULL,
  `Timestamp` DATETIME NOT NULL,
  `Author` BIGINT(20) UNSIGNED NOT NULL,
  `Message` VARCHAR(2000) NOT NULL,
  `Channel` BIGINT(20) UNSIGNED NOT NULL,
  `Everyone` BIT(1) NOT NULL,
  `Guild` BIGINT(20) UNSIGNED NOT NULL,
  PRIMARY KEY (`ID`, `Timestamp`),
  INDEX `INDEX_TIMESTAMP` (`Timestamp`),
  INDEX `INDEX_CHANNEL` (`Channel`),
  INDEX `CHATLOG_USERS` (`Author`),
  CONSTRAINT `EDITLOG_CHATLOG` FOREIGN KEY (`ID`) REFERENCES `chatlog` (`ID`),
  CONSTRAINT `editlog_ibfk_1` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=COMPACT COMMENT='A log of all the messages from all the chatrooms.'//

CREATE FUNCTION `GetMarkov`(`_prev` BIGINT) RETURNS bigint(20)
    READS SQL DATA
BEGIN

DECLARE n, c, t, weight_sum, weight INT;
DECLARE cur1 CURSOR FOR SELECT Next, Count FROM markov_transcripts_map WHERE Prev = _prev;
SET weight_sum = (SELECT SUM(Count) FROM markov_transcripts_map WHERE Prev = _prev);
SET weight = ROUND(((weight_sum - 1) * RAND() + 1), 0);
SET t = 0;

OPEN cur1;

WHILE t < weight DO
FETCH cur1 INTO n, c;
SET t = t + c;
END WHILE;

return n;

/*SET @next = 0;
SET @i = 0;
SET @s = '';

SET @next = (SELECT t1.Next
FROM markov_transcripts_map t1, markov_transcripts_map t2
WHERE t1.Prev = 0 AND t2.Prev = 0 AND t1.Next >= t2.Next
GROUP BY t1.Next
HAVING SUM(t2.Count) >= @weight
ORDER BY t1.Next
LIMIT 1);*/

END//

CREATE FUNCTION `GetMarkov2`(`_prev` BIGINT, `_prev2` BIGINT) RETURNS bigint(20)
    READS SQL DATA
BEGIN

DECLARE n, c, t, weight_sum, weight INT;
DECLARE cur1 CURSOR FOR SELECT Next, Count FROM markov_transcripts_map WHERE Prev = _prev AND Prev2 = _prev2;
SET weight_sum = (SELECT SUM(Count) FROM markov_transcripts_map WHERE Prev = _prev AND Prev2 = _prev2);
SET weight = ROUND(((weight_sum - 1) * RAND() + 1), 0);
SET t = 0;

OPEN cur1;

WHILE t < weight DO
FETCH cur1 INTO n, c;
SET t = t + c;
END WHILE;

return n;

END//

CREATE FUNCTION `GetMarkovLine`(`_prev` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
    READS SQL DATA
BEGIN

DECLARE line VARCHAR(1024) DEFAULT '|';
SET @prev = _prev;
IF NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev) THEN
	RETURN '|';
END IF;

SET @prev = GetMarkov(@prev);
SET @actionid = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = 'ACTION');
SET @speakerid = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @prev);
SET @speaker = (SELECT Speaker FROM markov_transcripts_speaker WHERE ID = @speakerid);
SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
SET @max = 0;

IF @speaker = 'ACTION' THEN
	IF @phrase = '' THEN RETURN CONCAT('|', @prev); END IF;
	SET line = CONCAT('[', @phrase);
ELSE
	SET line = CONCAT('**', @speaker, ':** ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
END IF;

markov_loop: LOOP
	IF @max > 300 OR NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev) THEN LEAVE markov_loop; END IF;
	SET @max = @max + 1;
	SET @capitalize = @phrase = '.' OR @phrase = '!' OR @phrase = '?';
	
	SET @next = GetMarkov(@prev);
	SET @ns = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
	IF @speakerid != @ns THEN LEAVE markov_loop; END IF;
	SET @prev = @next;
	
	SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
	IF @phrase = '.' OR @phrase = '!' OR @phrase = '?' OR @phrase = ',' THEN
		SET line = CONCAT(line, @phrase);
	ELSE
		IF @capitalize THEN
			SET line = CONCAT(line, ' ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
		ELSE
			SET line = CONCAT(line, ' ', @phrase);
		END IF;
	END IF;
END LOOP markov_loop;

IF @speaker = 'ACTION' THEN
	SET line = CONCAT(line, ']');
END IF;
RETURN CONCAT(line, '|', @prev);

END//

CREATE FUNCTION `GetMarkovLine2`(`_prev` BIGINT, `_prev2` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
    READS SQL DATA
BEGIN

DECLARE line VARCHAR(1024) DEFAULT '|';
SET @prev = _prev;
SET @prev2 = _prev2;
IF NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev AND Prev2 = @prev2) THEN
	RETURN '|';
END IF;

SET @next = GetMarkov2(@prev, @prev2);
SET @actionid = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = 'ACTION');
SET @speakerid = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
SET @speaker = (SELECT Speaker FROM markov_transcripts_speaker WHERE ID = @speakerid);
SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @next);
SET @max = 0;
SET @prev2 = @prev;
SET @prev = @next;

IF @speaker = 'ACTION' THEN
	IF @phrase = '' THEN RETURN CONCAT('|', @prev, '|', @prev2); END IF;
	SET line = CONCAT('[', @phrase);
ELSE
	SET line = CONCAT('**', @speaker, ':** ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
END IF;

markov_loop: LOOP
	IF @max > 300 OR NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev AND Prev2 = @prev2) THEN LEAVE markov_loop; END IF;
	SET @max = @max + 1;
	SET @capitalize = @phrase = '.' OR @phrase = '!' OR @phrase = '?';
	
	SET @next = GetMarkov2(@prev, @prev2);
	SET @ns = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
	IF @speakerid != @ns THEN LEAVE markov_loop; END IF;
  SET @prev2 = @prev;
	SET @prev = @next;
	
	SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
	IF @phrase = '.' OR @phrase = '!' OR @phrase = '?' OR @phrase = ',' THEN
		SET line = CONCAT(line, @phrase);
	ELSE
		IF @capitalize THEN
			SET line = CONCAT(line, ' ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
		ELSE
			SET line = CONCAT(line, ' ', @phrase);
		END IF;
	END IF;
END LOOP markov_loop;

IF @speaker = 'ACTION' THEN
	SET line = CONCAT(line, ']');
END IF;
RETURN CONCAT(line, '|', @prev, '|', @prev2);

END//

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
-- Dumping structure for table sweetiebot.markov_transcripts_speaker
CREATE TABLE IF NOT EXISTS `markov_transcripts_speaker` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Speaker` varchar(64) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `INDEX_SPEAKER` (`Speaker`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.markov_transcripts
CREATE TABLE IF NOT EXISTS `markov_transcripts` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `SpeakerID` bigint(20) unsigned NOT NULL DEFAULT 0,
  `Phrase` varchar(64) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `INDEX_SPEAKER_PHRASE` (`SpeakerID`,`Phrase`),
  CONSTRAINT `FK_TRANSCRIPTS_SPEAKER` FOREIGN KEY (`SpeakerID`) REFERENCES `markov_transcripts_speaker` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.markov_transcripts_map
CREATE TABLE IF NOT EXISTS `markov_transcripts_map` (
  `Prev` bigint(20) unsigned NOT NULL,
  `Prev2` bigint(20) unsigned NOT NULL,
  `Next` bigint(20) unsigned NOT NULL,
  `Count` int(10) unsigned NOT NULL DEFAULT 1,
  PRIMARY KEY (`Prev`,`Next`,`Prev2`),
  KEY `FK_NEXT` (`Next`),
  KEY `INDEX_PREV` (`Prev`),
  KEY `FK_PREV2` (`Prev2`),
  CONSTRAINT `FK_NEXT` FOREIGN KEY (`Next`) REFERENCES `markov_transcripts` (`ID`),
  CONSTRAINT `FK_PREV` FOREIGN KEY (`Prev`) REFERENCES `markov_transcripts` (`ID`),
  CONSTRAINT `FK_PREV2` FOREIGN KEY (`Prev2`) REFERENCES `markov_transcripts` (`ID`)
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

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.polls
CREATE TABLE IF NOT EXISTS `polls` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Guild` bigint(20) unsigned NOT NULL,
  `Name` varchar(50) NOT NULL,
  `Description` varchar(2048) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `Index 2` (`Name`,`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.polloptions
CREATE TABLE IF NOT EXISTS `polloptions` (
  `Poll` bigint(20) unsigned NOT NULL,
  `Index` bigint(20) unsigned NOT NULL,
  `Option` varchar(128) NOT NULL,
  PRIMARY KEY (`Poll`,`Index`),
  UNIQUE KEY `OPTION_INDEX` (`Option`,`Poll`),
  KEY `POLL_INDEX` (`Poll`),
  CONSTRAINT `FK_options_polls` FOREIGN KEY (`Poll`) REFERENCES `polls` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for view sweetiebot.randomwords
-- Creating temporary table to overcome VIEW dependency errors
CREATE TABLE `randomwords` (
	`Phrase` VARCHAR(64) NOT NULL COLLATE 'utf8mb4_general_ci'
) ENGINE=MyISAM//

CREATE PROCEDURE `RemoveGuild`(
	IN `_guild` BIGINT UNSIGNED
)
    MODIFIES SQL DATA
BEGIN

DELETE FROM `members` WHERE Guild = _guild;
DELETE FROM `polls` WHERE Guild = _guild;
DELETE FROM `schedule` WHERE Guild = _guild;
DELETE FROM `chatlog` WHERE Guild = _guild;
DELETE FROM `debuglog` WHERE Guild = _guild;
DELETE FROM `editlog` WHERE Guild = _guild;
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

CREATE PROCEDURE `ResetMarkov`()
    MODIFIES SQL DATA
BEGIN

SET foreign_key_checks = 0;
DELETE FROM markov_transcripts;
DELETE FROM markov_transcripts_speaker;
DELETE FROM markov_transcripts_map;
SET foreign_key_checks = 1;
ALTER TABLE `markov_transcripts` AUTO_INCREMENT=0;
ALTER TABLE `markov_transcripts_speaker` AUTO_INCREMENT=0;
INSERT INTO markov_transcripts_speaker (Speaker)
VALUES ('ACTION');
INSERT INTO markov_transcripts (ID, SpeakerID, Phrase)
VALUES (0, 1, '');
UPDATE markov_transcripts SET ID = 0 WHERE ID = 1;

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
  `Speaker` varchar(64) NOT NULL,
  `Text` varchar(2000) NOT NULL,
  PRIMARY KEY (`Season`,`Episode`,`Line`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Data exporting was unselected.
-- Dumping structure for table sweetiebot.votes
CREATE TABLE IF NOT EXISTS `votes` (
  `Poll` bigint(20) unsigned NOT NULL,
  `User` bigint(20) unsigned NOT NULL,
  `Option` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`Poll`,`User`),
  KEY `FK_votes_users` (`User`),
  KEY `FK_votes_options` (`Poll`,`Option`),
  CONSTRAINT `FK_votes_options` FOREIGN KEY (`Poll`, `Option`) REFERENCES `polloptions` (`Poll`, `Index`),
  CONSTRAINT `FK_votes_users` FOREIGN KEY (`User`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

-- Dumping structure for trigger sweetiebot.chatlog_before_update
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW INSERT INTO editlog (ID, `Timestamp`, Author, Message, Channel, Everyone, Guild)
VALUES (OLD.ID, OLD.`Timestamp`, OLD.Author, OLD.Message, OLD.Channel, OLD.Everyone, OLD.Guild)
ON DUPLICATE KEY UPDATE `Timestamp` = OLD.`Timestamp`, Message = OLD.Message//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.itemtags_after_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `itemtags_after_delete` AFTER DELETE ON `itemtags` FOR EACH ROW BEGIN

IF (SELECT COUNT(*) FROM itemtags WHERE Item = OLD.Item) = 0 THEN
DELETE FROM items WHERE ID = OLD.Item;
END IF;

END//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.polloptions_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `polloptions_before_delete` BEFORE DELETE ON `polloptions` FOR EACH ROW DELETE FROM votes WHERE Poll = OLD.Poll AND `Option` = OLD.`Index`//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.polls_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `polls_before_delete` BEFORE DELETE ON `polls` FOR EACH ROW DELETE FROM polloptions WHERE Poll = OLD.ID//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.tags_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `tags_before_delete` BEFORE DELETE ON `tags` FOR EACH ROW DELETE FROM itemtags WHERE Tag = OLD.ID//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for trigger sweetiebot.users_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION'//
CREATE TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members table
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM chatlog WHERE `Author` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;
DELETE FROM editlog WHERE `Author` = OLD.ID;
DELETE FROM votes WHERE `User` = OLD.ID;

END//
SET SQL_MODE=@OLDTMP_SQL_MODE//

-- Dumping structure for view sweetiebot.randomwords
-- Removing temporary table and create final VIEW structure
DROP TABLE IF EXISTS `randomwords`;
CREATE ALGORITHM=MERGE VIEW `randomwords` AS select `markov_transcripts`.`Phrase` AS `Phrase` from `markov_transcripts` where `markov_transcripts`.`Phrase` <> '.' and `markov_transcripts`.`Phrase` <> '!' and `markov_transcripts`.`Phrase` <> '?' and `markov_transcripts`.`Phrase` <> 'the' and `markov_transcripts`.`Phrase` <> 'of' and `markov_transcripts`.`Phrase` <> 'a' and `markov_transcripts`.`Phrase` <> 'to' and `markov_transcripts`.`Phrase` <> 'too' and `markov_transcripts`.`Phrase` <> 'as' and `markov_transcripts`.`Phrase` <> 'at' and `markov_transcripts`.`Phrase` <> 'an' and `markov_transcripts`.`Phrase` <> 'am' and `markov_transcripts`.`Phrase` <> 'and' and `markov_transcripts`.`Phrase` <> 'be' and `markov_transcripts`.`Phrase` <> 'he' and `markov_transcripts`.`Phrase` <> 'she' and `markov_transcripts`.`Phrase` <> '' 
 WITH LOCAL CHECK OPTION //
