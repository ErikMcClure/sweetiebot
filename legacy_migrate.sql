DELIMITER //

CREATE TABLE IF NOT EXISTS `items` (
	`ID` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
	`Content` VARCHAR(500) NOT NULL,
	PRIMARY KEY (`ID`),
	INDEX `CONTENT_INDEX` (`Content`(191))
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
AUTO_INCREMENT=1
//

CREATE TABLE IF NOT EXISTS `tags` (
	`ID` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
	`Name` VARCHAR(50) NOT NULL,
	`Guild` BIGINT(20) UNSIGNED NOT NULL,
	PRIMARY KEY (`ID`),
	UNIQUE INDEX `UNIQUE_NAME` (`Name`, `Guild`)
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
AUTO_INCREMENT=1
//

CREATE TABLE IF NOT EXISTS `polloptions` (
  `Poll` bigint(20) unsigned NOT NULL,
  `Index` bigint(20) unsigned NOT NULL,
  `Option` varchar(128) NOT NULL,
  PRIMARY KEY (`Poll`,`Index`),
  UNIQUE KEY `OPTION_INDEX` (`Option`,`Poll`),
  KEY `POLL_INDEX` (`Poll`),
  CONSTRAINT `FK_options_polls` FOREIGN KEY (`Poll`) REFERENCES `polls` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
//

CREATE TABLE IF NOT EXISTS `polls` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Guild` bigint(20) unsigned NOT NULL,
  `Name` varchar(50) NOT NULL,
  `Description` varchar(2048) NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `Index 2` (`Name`,`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
//

CREATE TABLE IF NOT EXISTS `tags` (
  `ID` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `Name` varchar(50) NOT NULL,
  `Guild` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `UNIQUE_NAME` (`Name`,`Guild`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
//

ALTER TABLE `members`
	DROP COLUMN IF EXISTS `LastNickChange`//
  
IF NOT EXISTS (SELECT * FROM `information_schema`.`COLUMNS`
WHERE `TABLE_NAME` = "members" AND `COLUMN_NAME` = "FirstMessage") 
THEN
   ALTER TABLE `members` ADD `FirstMessage` datetime DEFAULT NULL;
END IF//

ALTER TABLE `users`
	DROP COLUMN IF EXISTS `Email`,
	DROP COLUMN IF EXISTS `Timezone`,
	DROP COLUMN IF EXISTS `Verified`//

IF NOT EXISTS (SELECT * FROM `information_schema`.`COLUMNS`
WHERE `TABLE_NAME` = "users" AND `COLUMN_NAME` = "Discriminator") 
THEN
   ALTER TABLE `users` ADD `Discriminator` int(10) unsigned NOT NULL DEFAULT '0';
END IF//
	
DROP PROCEDURE IF EXISTS `SawUser`//
DROP TRIGGER IF EXISTS `members_before_update`//

DROP PROCEDURE IF EXISTS `AddChat`//
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

INSERT INTO users (ID, Username, Avatar, LastSeen, LastNameChange) 
VALUES (_author, _username, '', UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE LastSeen=UTC_TIMESTAMP();

INSERT IGNORE INTO aliases (`User`, Alias, Duration, `Timestamp`)
VALUES (_id, _username, 0, UTC_TIMESTAMP());

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Guild)
VALUES (_id, _author, _message, UTC_TIMESTAMP(), _channel, _guild)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = UTC_TIMESTAMP();

END//

DROP FUNCTION IF EXISTS `AddItem`//
CREATE DEFINER=`root`@`localhost` FUNCTION `AddItem`(`_content` VARCHAR(500)) RETURNS bigint(20)
    MODIFIES SQL DATA
BEGIN
SET @id = (SELECT ID FROM items WHERE Content = _content);

IF @id IS NULL THEN
	INSERT INTO items (Content) VALUES (_content);
	RETURN LAST_INSERT_ID();
END IF;

RETURN @id;
END//

DROP FUNCTION IF EXISTS `AddMarkov`//
CREATE DEFINER=`root`@`localhost` FUNCTION `AddMarkov`(`_prev` BIGINT, `_prev2` BIGINT, `_speaker` VARCHAR(128), `_phrase` VARCHAR(64)) RETURNS bigint(20)
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

DROP PROCEDURE IF EXISTS `AddMember`//
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddMember`(IN `_id` BIGINT, IN `_guild` BIGINT, IN `_firstseen` DATETIME, IN `_nickname` VARCHAR(128))
	LANGUAGE SQL
	NOT DETERMINISTIC
	MODIFIES SQL DATA
	SQL SECURITY DEFINER
	COMMENT ''
INSERT INTO members (ID, Guild, FirstSeen, Nickname)
VALUES (_id, _guild, _firstseen, _nickname)
ON DUPLICATE KEY UPDATE
FirstSeen=GetMinDate(_firstseen,FirstSeen), Nickname=_nickname//

DROP PROCEDURE IF EXISTS `AddUser`//
CREATE PROCEDURE `AddUser`(
	IN `_id` BIGINT,
	IN `_username` VARCHAR(128),
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

DROP TRIGGER IF EXISTS `chatlog_before_delete`//
CREATE TRIGGER `chatlog_before_delete` BEFORE DELETE ON `chatlog` FOR EACH ROW DELETE FROM editlog WHERE ID = OLD.ID//

DROP TRIGGER IF EXISTS `chatlog_before_update`//
CREATE TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW INSERT INTO editlog (ID, `Timestamp`, Author, Message, Channel, Guild)
VALUES (OLD.ID, OLD.`Timestamp`, OLD.Author, OLD.Message, OLD.Channel, OLD.Guild)
ON DUPLICATE KEY UPDATE `Timestamp` = OLD.`Timestamp`, Message = OLD.Message//

CREATE TABLE IF NOT EXISTS `itemtags` (
  `Item` bigint(20) unsigned NOT NULL,
  `Tag` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`Item`,`Tag`),
  KEY `FK_itemtags_tags` (`Tag`),
  CONSTRAINT `FK_itemtags_items` FOREIGN KEY (`Item`) REFERENCES `items` (`ID`),
  CONSTRAINT `FK_itemtags_tags` FOREIGN KEY (`Tag`) REFERENCES `tags` (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4//

DROP TRIGGER IF EXISTS `itemtags_after_delete`//
CREATE TRIGGER `itemtags_after_delete` AFTER DELETE ON `itemtags` FOR EACH ROW BEGIN

IF (SELECT COUNT(*) FROM itemtags WHERE Item = OLD.Item) = 0 THEN
	DELETE FROM items WHERE ID = OLD.Item;
END IF;

END//

DROP TRIGGER IF EXISTS `polloptions_before_delete`//
CREATE TRIGGER `polloptions_before_delete` BEFORE DELETE ON `polloptions` FOR EACH ROW DELETE FROM votes WHERE Poll = OLD.Poll AND `Option` = OLD.`Index`//

DROP TRIGGER IF EXISTS `polls_before_delete`//
CREATE TRIGGER `polls_before_delete` BEFORE DELETE ON `polls` FOR EACH ROW DELETE FROM polloptions WHERE Poll = OLD.ID//

DROP TRIGGER IF EXISTS `tags_before_delete`//
CREATE TRIGGER `tags_before_delete` BEFORE DELETE ON `tags` FOR EACH ROW DELETE FROM itemtags WHERE Tag = OLD.ID//

DROP TRIGGER IF EXISTS `users_before_update`//

ALTER TABLE `chatlog`
	DROP COLUMN IF EXISTS `Everyone`//

ALTER TABLE `editlog`
	DROP COLUMN IF EXISTS `Everyone`//
  
DROP PROCEDURE IF EXISTS `RemoveGuild`//
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
