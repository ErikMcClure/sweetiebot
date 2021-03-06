DELIMITER //

DROP PROCEDURE IF EXISTS `AddChat` //
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddChat`(IN `_id` BIGINT, IN `_author` BIGINT, IN `_message` VARCHAR(2000), IN `_channel` BIGINT, IN `_everyone` BIT, IN `_guild` BIGINT)
	LANGUAGE SQL
	NOT DETERMINISTIC
	MODIFIES SQL DATA
	SQL SECURITY DEFINER
	COMMENT ''
BEGIN

INSERT INTO users (ID, Username, Avatar, LastSeen, LastNameChange) 
VALUES (_author, '', '', UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE LastSeen=UTC_TIMESTAMP();

INSERT INTO chatlog (ID, Author, Message, Timestamp, Channel, Everyone, Guild)
VALUES (_id, _author, _message, UTC_TIMESTAMP(), _channel, _everyone, _guild)
ON DUPLICATE KEY UPDATE /* This prevents a race condition from causing a serious error */
Message = _message COLLATE 'utf8mb4_general_ci', Timestamp = UTC_TIMESTAMP(), Everyone=_everyone;

END//

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

CREATE TABLE IF NOT EXISTS `itemtags` (
	`Item` BIGINT(20) UNSIGNED NOT NULL,
	`Tag` BIGINT(20) UNSIGNED NOT NULL,
	PRIMARY KEY (`Item`, `Tag`),
	INDEX `FK_itemtags_tags` (`Tag`),
	CONSTRAINT `FK_itemtags_items` FOREIGN KEY (`Item`) REFERENCES `items` (`ID`),
	CONSTRAINT `FK_itemtags_tags` FOREIGN KEY (`Tag`) REFERENCES `tags` (`ID`)
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
//

DROP FUNCTION IF EXISTS `AddItem`//
CREATE DEFINER=`root`@`localhost` FUNCTION `AddItem`(`_content` VARCHAR(500))
	RETURNS bigint(20)
	LANGUAGE SQL
	NOT DETERMINISTIC
	MODIFIES SQL DATA
	SQL SECURITY DEFINER
	COMMENT ''
BEGIN
SET @id = (SELECT ID FROM items WHERE Content = _content);

IF @id IS NULL THEN
	INSERT INTO items (Content) VALUES (_content);
	RETURN LAST_INSERT_ID();
END IF;

RETURN @id;
END//

DROP TRIGGER IF EXISTS `itemtags_after_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `itemtags_after_delete` AFTER DELETE ON `itemtags` FOR EACH ROW BEGIN

IF (SELECT COUNT(*) FROM itemtags WHERE Item = OLD.Item) = 0 THEN
	DELETE FROM items WHERE ID = OLD.Item;
END IF;

END//

DROP TRIGGER IF EXISTS `tags_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `tags_before_delete` BEFORE DELETE ON `tags` FOR EACH ROW DELETE FROM itemtags WHERE Tag = OLD.ID//

DROP TRIGGER IF EXISTS `polls_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `polls_before_delete` BEFORE DELETE ON `polls` FOR EACH ROW DELETE FROM polloptions WHERE Poll = OLD.ID//

DROP TRIGGER IF EXISTS `polloptions_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `polloptions_before_delete` BEFORE DELETE ON `polloptions` FOR EACH ROW DELETE FROM votes WHERE Poll = OLD.Poll AND `Option` = OLD.`Index`//

DROP TRIGGER IF EXISTS `chatlog_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `chatlog_before_delete` BEFORE DELETE ON `chatlog` FOR EACH ROW DELETE FROM editlog WHERE ID = OLD.ID//

DROP TRIGGER IF EXISTS `chatlog_before_update`//
CREATE DEFINER=`root`@`localhost` TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW INSERT INTO editlog (ID, Author, Message, Timestamp, Channel, Everyone, Guild)
VALUES (OLD.ID, OLD.Author, OLD.Message, OLD.Timestamp, OLD.Channel, OLD.Everyone, OLD.Guild)
ON DUPLICATE KEY UPDATE ID = OLD.ID//

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

ALTER TABLE `members`
	DROP COLUMN IF EXISTS `LastNickChange`//

DROP PROCEDURE IF EXISTS `AddUser`//
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddUser`(IN `_id` BIGINT, IN `_username` VARCHAR(512), IN `_discriminator` INT, IN `_avatar` VARCHAR(512), IN `_isonline` BIT)
	LANGUAGE SQL
	NOT DETERMINISTIC
	MODIFIES SQL DATA
	SQL SECURITY DEFINER
	COMMENT ''
INSERT INTO users (ID, Username, Discriminator, Avatar, LastSeen, LastNameChange) 
VALUES (_id, _username, _discriminator, _avatar, UTC_TIMESTAMP(), UTC_TIMESTAMP()) 
ON DUPLICATE KEY UPDATE 
Username=_username, Discriminator=_discriminator, Avatar=_avatar, LastSeen=IF(_isonline > 0, UTC_TIMESTAMP(), LastSeen)//
  
DROP TRIGGER IF EXISTS `users_before_update`//
CREATE DEFINER=`root`@`localhost` TRIGGER `users_before_update` BEFORE UPDATE ON `users` FOR EACH ROW BEGIN

IF NEW.Username = '' THEN
SET NEW.Username = OLD.Username;
END IF;

IF NEW.Discriminator = 0 THEN
SET NEW.Discriminator = OLD.Discriminator;
END IF;

IF NEW.Avatar = '' THEN
SET NEW.Avatar = OLD.Avatar;
END IF;

IF NEW.Username != OLD.Username THEN
SET NEW.LastNameChange = UTC_TIMESTAMP();
SET @diff = UNIX_TIMESTAMP(NEW.LastNameChange) - UNIX_TIMESTAMP(OLD.LastNameChange);
INSERT INTO aliases (User, Alias, Duration)
VALUES (OLD.ID, OLD.Username, @diff)
ON DUPLICATE KEY UPDATE Duration = Duration + @diff;
END IF;

END//
  
ALTER TABLE `users`
	DROP COLUMN IF EXISTS `Email`,
	DROP COLUMN IF EXISTS `Verified`//

DROP PROCEDURE IF EXISTS `SawUser`//