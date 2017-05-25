-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               10.1.19-MariaDB - mariadb.org binary distribution
-- Server OS:                    Win64
-- HeidiSQL Version:             9.3.0.4984
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
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddChat`(IN `_id` BIGINT, IN `_author` BIGINT, IN `_message` VARCHAR(2000), IN `_channel` BIGINT, IN `_everyone` BIT, IN `_guild` BIGINT)
    DETERMINISTIC
BEGIN

CALL SawUser(_author);

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Everyone, Guild)
VALUES (_id, _author, _message, UTC_TIMESTAMP(), _channel, _everyone, _guild)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = UTC_TIMESTAMP(), Everyone=_everyone;

END//
DELIMITER ;


-- Dumping structure for function sweetiebot.AddMarkov
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `AddMarkov`(`_prev` BIGINT, `_prev2` BIGINT, `_speaker` VARCHAR(64), `_phrase` VARCHAR(64)) RETURNS bigint(20)
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
DELIMITER ;


-- Dumping structure for procedure sweetiebot.AddMember
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddMember`(IN `_id` BIGINT, IN `_guild` BIGINT, IN `_firstseen` DATETIME, IN `_nickname` VARCHAR(128))
INSERT INTO members (ID, Guild, FirstSeen, Nickname, LastNickChange)
VALUES (_id, _guild, _firstseen, _nickname, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE
FirstSeen=GetMinDate(_firstseen,FirstSeen), Nickname=_nickname//
DELIMITER ;


-- Dumping structure for procedure sweetiebot.AddUser
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddUser`(IN `_id` BIGINT, IN `_email` VARCHAR(512), IN `_username` VARCHAR(512), IN `_discriminator` INT, IN `_avatar` VARCHAR(512), IN `_verified` BIT, IN `_isonline` BIT)
    DETERMINISTIC
INSERT INTO users (ID, Email, Username, Discriminator, Avatar, Verified, LastSeen, LastNameChange) 
VALUES (_id, _email, _username, _discriminator, _avatar, _verified, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE 
Username=_username, Discriminator=_discriminator, Avatar=_avatar, Email = _email, Verified=_verified, LastSeen=IF(_isonline > 0, UTC_TIMESTAMP(), LastSeen)//
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='A log of all the messages from all the chatrooms.';

-- Data exporting was unselected.


-- Dumping structure for event sweetiebot.CleanChatlog
DELIMITER //
CREATE DEFINER=`root`@`localhost` EVENT `CleanChatlog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:04:34' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM chatlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
END//
DELIMITER ;


-- Dumping structure for event sweetiebot.CleanDebugLog
DELIMITER //
CREATE DEFINER=`root`@`localhost` EVENT `CleanDebugLog` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:30:36' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
DELETE FROM debuglog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 8 DAY);
END//
DELIMITER ;


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
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_TIMESTAMP` (`Timestamp`),
  KEY `INDEX_CHANNEL` (`Channel`),
  KEY `CHATLOG_USERS` (`Author`),
  CONSTRAINT `EDITLOG_CHATLOG` FOREIGN KEY (`ID`) REFERENCES `chatlog` (`ID`),
  CONSTRAINT `editlog_ibfk_1` FOREIGN KEY (`Author`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=COMPACT COMMENT='A log of all the messages from all the chatrooms.';

-- Data exporting was unselected.


-- Dumping structure for function sweetiebot.GetMarkov
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `GetMarkov`(`_prev` BIGINT) RETURNS bigint(20)
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
DELIMITER ;


-- Dumping structure for function sweetiebot.GetMarkov2
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `GetMarkov2`(`_prev` BIGINT, `_prev2` BIGINT) RETURNS bigint(20)
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
DELIMITER ;


-- Dumping structure for function sweetiebot.GetMarkovLine
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `GetMarkovLine`(`_prev` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
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
DELIMITER ;


-- Dumping structure for function sweetiebot.GetMarkovLine2
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `GetMarkovLine2`(`_prev` BIGINT, `_prev2` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
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
DELIMITER ;


-- Dumping structure for function sweetiebot.GetMinDate
DELIMITER //
CREATE DEFINER=`root`@`localhost` FUNCTION `GetMinDate`(`date1` DATETIME, `date2` DATETIME) RETURNS datetime
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
DELIMITER ;


-- Dumping structure for table sweetiebot.markov_transcripts
CREATE TABLE IF NOT EXISTS `markov_transcripts` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `SpeakerID` bigint(20) unsigned NOT NULL DEFAULT '0',
  `Phrase` varchar(64) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `INDEX_SPEAKER_PHRASE` (`SpeakerID`,`Phrase`),
  CONSTRAINT `FK_TRANSCRIPTS_SPEAKER` FOREIGN KEY (`SpeakerID`) REFERENCES `markov_transcripts_speaker` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.markov_transcripts_map
CREATE TABLE IF NOT EXISTS `markov_transcripts_map` (
  `Prev` bigint(20) unsigned NOT NULL,
  `Prev2` bigint(20) unsigned NOT NULL,
  `Next` bigint(20) unsigned NOT NULL,
  `Count` int(10) unsigned NOT NULL DEFAULT '1',
  PRIMARY KEY (`Prev`,`Next`,`Prev2`),
  KEY `FK_NEXT` (`Next`),
  KEY `INDEX_PREV` (`Prev`),
  KEY `FK_PREV2` (`Prev2`),
  CONSTRAINT `FK_NEXT` FOREIGN KEY (`Next`) REFERENCES `markov_transcripts` (`ID`),
  CONSTRAINT `FK_PREV` FOREIGN KEY (`Prev`) REFERENCES `markov_transcripts` (`ID`),
  CONSTRAINT `FK_PREV2` FOREIGN KEY (`Prev2`) REFERENCES `markov_transcripts` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.markov_transcripts_speaker
CREATE TABLE IF NOT EXISTS `markov_transcripts_speaker` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Speaker` varchar(64) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `INDEX_SPEAKER` (`Speaker`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.members
CREATE TABLE IF NOT EXISTS `members` (
  `ID` bigint(20) unsigned NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  `FirstSeen` datetime NOT NULL,
  `Nickname` varchar(128) NOT NULL DEFAULT '',
  `LastNickChange` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`ID`,`Guild`),
  KEY `INDEX_NICKNAME` (`Nickname`),
  CONSTRAINT `FK_members_users` FOREIGN KEY (`ID`) REFERENCES `users` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.polls
CREATE TABLE IF NOT EXISTS `polls` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Guild` bigint(20) unsigned NOT NULL,
  `Name` varchar(50) NOT NULL,
  `Description` varchar(2048) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `Index 2` (`Name`,`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for view sweetiebot.randomwords
-- Creating temporary table to overcome VIEW dependency errors
CREATE TABLE `randomwords` (
	`Phrase` VARCHAR(64) NOT NULL COLLATE 'utf8mb4_general_ci'
) ENGINE=MyISAM;


-- Dumping structure for procedure sweetiebot.RemoveSchedule
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `RemoveSchedule`(IN `_id` BIGINT)
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
DELIMITER ;


-- Dumping structure for procedure sweetiebot.ResetMarkov
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `ResetMarkov`()
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
DELIMITER ;


-- Dumping structure for procedure sweetiebot.SawUser
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `SawUser`(IN `_id` BIGINT)
INSERT INTO users (ID, Email, Username, Avatar, Verified, LastSeen, LastNameChange) 
VALUES (_id, '', '', '', 0, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE LastSeen=UTC_TIMESTAMP()//
DELIMITER ;


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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.timezones
CREATE TABLE IF NOT EXISTS `timezones` (
  `Location` varchar(40) NOT NULL,
  `Offset` int(11) NOT NULL,
  `DST` int(11) NOT NULL,
  PRIMARY KEY (`Location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.transcripts
CREATE TABLE IF NOT EXISTS `transcripts` (
  `Season` int(10) unsigned NOT NULL,
  `Episode` int(10) unsigned NOT NULL,
  `Line` int(10) unsigned NOT NULL,
  `Speaker` varchar(64) NOT NULL,
  `Text` varchar(2000) NOT NULL,
  PRIMARY KEY (`Season`,`Episode`,`Line`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for table sweetiebot.users
CREATE TABLE IF NOT EXISTS `users` (
  `ID` bigint(20) unsigned NOT NULL,
  `Email` varchar(512) NOT NULL DEFAULT '',
  `Username` varchar(128) NOT NULL DEFAULT '',
  `Discriminator` int(10) unsigned NOT NULL DEFAULT '0',
  `Avatar` varchar(512) NOT NULL DEFAULT '',
  `Verified` bit(1) NOT NULL DEFAULT b'0',
  `LastSeen` datetime NOT NULL,
  `LastNameChange` datetime NOT NULL,
  `Location` varchar(40) DEFAULT NULL,
  `DefaultServer` bigint(20) unsigned DEFAULT NULL,
  PRIMARY KEY (`ID`),
  KEY `INDEX_USERNAME` (`Username`),
  KEY `FK_Location_timezone` (`Location`),
  CONSTRAINT `FK_Location_timezone` FOREIGN KEY (`Location`) REFERENCES `timezones` (`Location`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Data exporting was unselected.


-- Dumping structure for trigger sweetiebot.chatlog_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `chatlog_before_delete` BEFORE DELETE ON `chatlog` FOR EACH ROW BEGIN
DELETE FROM editlog WHERE ID = OLD.ID;
END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.chatlog_before_update
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW BEGIN

INSERT INTO editlog (ID, Author, Message, Timestamp, Channel, Everyone, Guild)
VALUES (OLD.ID, OLD.Author, OLD.Message, OLD.Timestamp, OLD.Channel, OLD.Everyone, OLD.Guild)
ON DUPLICATE KEY UPDATE ID = OLD.ID;

END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.polloptions_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `polloptions_before_delete` BEFORE DELETE ON `polloptions` FOR EACH ROW BEGIN
DELETE FROM votes WHERE Poll = OLD.Poll AND `Option` = OLD.`Index`;
END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.polls_before_delete
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `polls_before_delete` BEFORE DELETE ON `polls` FOR EACH ROW BEGIN
DELETE FROM polloptions WHERE Poll = OLD.ID;
END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for trigger sweetiebot.users_before_update
SET @OLDTMP_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
DELIMITER //
CREATE TRIGGER `users_before_update` BEFORE UPDATE ON `users` FOR EACH ROW BEGIN

IF NEW.Username = '' THEN
SET NEW.Username = OLD.Username;
END IF;

IF NEW.Discriminator = 0 THEN
SET NEW.Discriminator = OLD.Discriminator;
END IF;

IF NEW.Avatar = '' THEN
SET NEW.Avatar = OLD.Avatar;
END IF;

IF NEW.Email = '' THEN
SET NEW.Email = OLD.Email;
END IF;

IF NEW.Username != OLD.Username THEN
SET NEW.LastNameChange = UTC_TIMESTAMP();
SET @diff = UNIX_TIMESTAMP(NEW.LastNameChange) - UNIX_TIMESTAMP(OLD.LastNameChange);
INSERT INTO aliases (User, Alias, Duration)
VALUES (OLD.ID, OLD.Username, @diff)
ON DUPLICATE KEY UPDATE Duration = Duration + @diff;
END IF;

END//
DELIMITER ;
SET SQL_MODE=@OLDTMP_SQL_MODE;


-- Dumping structure for view sweetiebot.randomwords
-- Removing temporary table and create final VIEW structure
DROP TABLE IF EXISTS `randomwords`;
CREATE ALGORITHM=MERGE DEFINER=`root`@`localhost` VIEW `randomwords` AS select `markov_transcripts`.`Phrase` AS `Phrase` from `markov_transcripts` where ((`markov_transcripts`.`Phrase` <> '.') and (`markov_transcripts`.`Phrase` <> '!') and (`markov_transcripts`.`Phrase` <> '?') and (`markov_transcripts`.`Phrase` <> 'the') and (`markov_transcripts`.`Phrase` <> 'of') and (`markov_transcripts`.`Phrase` <> 'a') and (`markov_transcripts`.`Phrase` <> 'to') and (`markov_transcripts`.`Phrase` <> 'too') and (`markov_transcripts`.`Phrase` <> 'as') and (`markov_transcripts`.`Phrase` <> 'at') and (`markov_transcripts`.`Phrase` <> 'an') and (`markov_transcripts`.`Phrase` <> 'am') and (`markov_transcripts`.`Phrase` <> 'and') and (`markov_transcripts`.`Phrase` <> 'be') and (`markov_transcripts`.`Phrase` <> 'he') and (`markov_transcripts`.`Phrase` <> 'she') and (`markov_transcripts`.`Phrase` <> '')) 
 WITH LOCAL CHECK OPTION ;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
