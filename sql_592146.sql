DELIMITER //

DROP TRIGGER IF EXISTS `users_before_delete`//
CREATE TRIGGER `users_before_delete` BEFORE DELETE ON `users` FOR EACH ROW BEGIN

-- Note: You cannot delete a user unless they have no entries in the members, editlog or chatlog tables
DELETE FROM aliases WHERE `User` = OLD.ID;
DELETE FROM debuglog WHERE `User` = OLD.ID;
DELETE FROM votes WHERE `User` = OLD.ID;

END//

ALTER EVENT `CleanUsers`
	ON SCHEDULE
		EVERY 1 DAY STARTS '2018-01-22 15:50:17'
	ON COMPLETION NOT PRESERVE
	ENABLE
	COMMENT ''
	DO BEGIN
DELETE FROM users WHERE `ID` NOT IN (SELECT DISTINCT ID FROM members) AND `ID` NOT IN (SELECT DISTINCT ID FROM chatlog) AND `ID` NOT IN (SELECT DISTINCT ID FROM editlog);
END//