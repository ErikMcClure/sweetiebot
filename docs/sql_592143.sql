DELIMITER //

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

DROP TRIGGER IF EXISTS `chatlog_before_update`//
CREATE TRIGGER `chatlog_before_update` BEFORE UPDATE ON `chatlog` FOR EACH ROW INSERT INTO editlog (ID, `Timestamp`, Author, Message, Channel, Guild)
VALUES (OLD.ID, OLD.`Timestamp`, OLD.Author, OLD.Message, OLD.Channel, OLD.Guild)
ON DUPLICATE KEY UPDATE `Timestamp` = OLD.`Timestamp`, Message = OLD.Message//

ALTER TABLE `chatlog`
	DROP COLUMN `Everyone`//

ALTER TABLE `editlog`
	DROP COLUMN `Everyone`//