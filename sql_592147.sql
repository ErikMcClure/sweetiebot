DELIMITER //

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
DELETE FROM `editlog` WHERE Guild = _guild;
DELETE FROM `tags` WHERE Guild = _guild;

END//

DROP TRIGGER IF EXISTS `users_before_delete`//
CREATE TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members, editlog or chatlog tables
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;
DELETE FROM votes WHERE `User` = OLD.ID;

END//

DROP TABLE IF EXISTS votes//
DROP TRIGGER IF EXISTS `polloptions_before_delete`//
DROP TABLE IF EXISTS polloptions//
DROP TRIGGER IF EXISTS `polls_before_delete`//
DROP TABLE IF EXISTS polls//
