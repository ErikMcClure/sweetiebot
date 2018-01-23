DELIMITER //

ALTER TABLE `aliases`
	DROP COLUMN IF EXISTS `ID`,
	DROP INDEX `ALIASES_ALIAS`,
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (`User`, `Alias`),
	ADD COLUMN `Timestamp` TIMESTAMP NOT NULL AFTER `Alias`,
	ADD INDEX `ALIASES_ALIAS` (`Alias`)//
	
DROP TRIGGER IF EXISTS `users_before_update`//

ALTER TABLE `members`
	ADD INDEX `INDEX_GUILD_FIRSTSEEN` (`Guild`, `FirstSeen`)//

DROP PROCEDURE IF EXISTS `AddUser`//
CREATE DEFINER=`root`@`localhost` PROCEDURE `AddUser`(
	IN `_id` BIGINT,
	IN `_username` VARCHAR(512),
	IN `_discriminator` INT,
	IN `_avatar` VARCHAR(512),
	IN `_isonline` BIT
)
LANGUAGE SQL
NOT DETERMINISTIC
MODIFIES SQL DATA
SQL SECURITY DEFINER
COMMENT ''
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

DROP TRIGGER IF EXISTS `users_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members table
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM chatlog WHERE `Author` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;
DELETE FROM editlog WHERE `Author` = OLD.ID;
DELETE FROM votes WHERE `User` = OLD.ID;

END//

DROP EVENT IF EXISTS `CleanUsers`//
CREATE DEFINER=`root`@`localhost` EVENT `CleanUsers`
	ON SCHEDULE
		EVERY 1 DAY STARTS '2018-01-23 03:24:17'
	ON COMPLETION PRESERVE
	ENABLE
	COMMENT ''
	DO BEGIN
DELETE FROM users WHERE `ID` NOT IN (SELECT DISTINCT ID FROM members);
END //

DROP PROCEDURE IF EXISTS `RemoveGuild`//
CREATE DEFINER=`root`@`localhost` PROCEDURE `RemoveGuild`(
	IN `_guild` BIGINT UNSIGNED

)
LANGUAGE SQL
NOT DETERMINISTIC
MODIFIES SQL DATA
SQL SECURITY DEFINER
COMMENT ''
BEGIN

DELETE FROM `members` WHERE Guild = _guild;
DELETE FROM `polls` WHERE Guild = _guild;
DELETE FROM `schedule` WHERE Guild = _guild;
DELETE FROM `chatlog` WHERE Guild = _guild;
DELETE FROM `debuglog` WHERE Guild = _guild;
DELETE FROM `editlog` WHERE Guild = _guild;
DELETE FROM `tags` WHERE Guild = _guild;

END//

DELETE FROM users WHERE `ID` NOT IN (SELECT DISTINCT ID FROM members)//

INSERT INTO aliases (`User`, `Alias`, `Duration`, `Timestamp`)
SELECT `ID`, `Username`, (UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(`LastNameChange`)), UTC_TIMESTAMP()
FROM users
ON DUPLICATE KEY UPDATE
`Duration` = (UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(`LastNameChange`)),
`Timestamp` = UTC_TIMESTAMP()//
