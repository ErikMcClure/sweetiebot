DELIMITER //

ALTER TABLE `aliases`
	CHANGE COLUMN `Timestamp` `Timestamp` DATETIME NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp() AFTER `Alias`//
