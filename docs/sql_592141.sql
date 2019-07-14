DELIMITER //

ALTER DEFINER=`root`@`localhost` EVENT `CleanChatlog`
	ON SCHEDULE
		EVERY 1 DAY STARTS '2016-01-29 17:04:34'
	ON COMPLETION NOT PRESERVE
	ENABLE
	COMMENT ''
	DO BEGIN
DELETE FROM editlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
DELETE FROM chatlog WHERE Timestamp < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY);
END//

DROP TRIGGER IF EXISTS `users_before_delete`//
CREATE DEFINER=`root`@`localhost` TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members table
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM editlog WHERE `Author` = OLD.ID;
DELETE FROM chatlog WHERE `Author` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;
DELETE FROM votes WHERE `User` = OLD.ID;

END//

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
DELETE FROM `editlog` WHERE Guild = _guild;
DELETE FROM `chatlog` WHERE Guild = _guild;
DELETE FROM `debuglog` WHERE Guild = _guild;
DELETE FROM `tags` WHERE Guild = _guild;

END//