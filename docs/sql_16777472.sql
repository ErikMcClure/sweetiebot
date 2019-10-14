DELIMITER //

DROP TRIGGER IF EXISTS `members_before_delete`//
CREATE TRIGGER `members_before_delete` BEFORE DELETE ON `members` FOR EACH ROW BEGIN

DELETE FROM `schedule` WHERE 
	CONCAT(OLD.ID, '') = `Data`
	AND (
    `Type` = 1
    OR `Type` = 4
    OR `Type` = 8
  );

DELETE FROM `schedule` WHERE 
	`Data` LIKE CONCAT(OLD.ID, '|%')
	AND (`Type` = 9 OR `Type` = 6);
	
END//