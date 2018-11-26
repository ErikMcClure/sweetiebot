DELIMITER //

DROP TABLE IF EXISTS markov_transcripts_map//
DROP TABLE IF EXISTS markov_transcripts//
DROP TABLE IF EXISTS markov_transcripts_speaker//
DROP FUNCTION IF EXISTS `GetMarkov`//
DROP FUNCTION IF EXISTS `GetMarkovLine`//
DROP FUNCTION IF EXISTS `GetMarkov2`//
DROP FUNCTION IF EXISTS `GetMarkovLine2`//
DROP PROCEDURE IF EXISTS `ResetMarkov`//
DROP FUNCTION IF EXISTS `AddMarkov`//
DROP VIEW IF EXISTS randomwords//

DROP TRIGGER IF EXISTS `chatlog_before_update`//
DROP TABLE IF EXISTS editlog//

DROP EVENT IF EXISTS `CleanChatlog`//
DROP EVENT IF EXISTS `CleanUsers`//
DROP EVENT IF EXISTS `CleanDebugLog`//
DROP EVENT IF EXISTS `CleanAliases`//
DROP EVENT IF EXISTS `Clean`//

CREATE EVENT `Clean` ON SCHEDULE EVERY 1 DAY STARTS '2016-01-29 17:04:34' ON COMPLETION NOT PRESERVE ENABLE DO BEGIN
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

ALTER TABLE `chatlog`
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (`ID`, `Timestamp`)//

DROP PROCEDURE IF EXISTS `RemoveGuild`//
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

DROP TRIGGER IF EXISTS `users_before_delete`//
CREATE TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members or chatlog tables
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;

END//

ALTER TABLE `users`
	DROP COLUMN IF EXISTS `Avatar`//
  
DROP PROCEDURE `AddChat`//
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

DROP PROCEDURE `AddUser`//
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