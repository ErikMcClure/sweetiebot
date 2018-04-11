DELIMITER //

ALTER TABLE `transcripts`
	ALTER `Season` DROP DEFAULT,
	ALTER `Episode` DROP DEFAULT,
	ALTER `Line` DROP DEFAULT,
	ALTER `Speaker` DROP DEFAULT//
ALTER TABLE `transcripts`
	MODIFY COLUMN `Season` TINYINT UNSIGNED NOT NULL FIRST,
	MODIFY COLUMN `Episode` TINYINT UNSIGNED NOT NULL AFTER `Season`,
	MODIFY COLUMN `Line` SMALLINT UNSIGNED NOT NULL AFTER `Episode`,
	MODIFY COLUMN `Speaker` VARCHAR(128) NOT NULL AFTER `Line`//
  
ALTER TABLE `markov_transcripts_map`
	ALTER `Prev` DROP DEFAULT,
	ALTER `Prev2` DROP DEFAULT,
	ALTER `Next` DROP DEFAULT//
ALTER TABLE `markov_transcripts_map`
	MODIFY COLUMN `Prev` INT UNSIGNED NOT NULL FIRST,
	MODIFY COLUMN `Prev2` INT UNSIGNED NOT NULL AFTER `Prev`,
	MODIFY COLUMN `Next` INT UNSIGNED NOT NULL AFTER `Prev2`,
	MODIFY COLUMN `Count` SMALLINT UNSIGNED NOT NULL DEFAULT '1' AFTER `Next`,
	DROP FOREIGN KEY IF EXISTS `FK_NEXT`,
	DROP FOREIGN KEY IF EXISTS `FK_PREV`,
	DROP FOREIGN KEY IF EXISTS `FK_PREV2`//
  
ALTER TABLE `markov_transcripts`
	MODIFY COLUMN `ID` INT UNSIGNED NOT NULL AUTO_INCREMENT FIRST,
	MODIFY COLUMN `SpeakerID` INT UNSIGNED NOT NULL DEFAULT '0' AFTER `ID`,
	DROP FOREIGN KEY IF EXISTS `FK_TRANSCRIPTS_SPEAKER`//

ALTER TABLE `markov_transcripts_speaker`
	MODIFY COLUMN `ID` INT UNSIGNED NOT NULL AUTO_INCREMENT FIRST//
  
ALTER TABLE `markov_transcripts_map`
	ADD CONSTRAINT `FK_NEXT` FOREIGN KEY (`Next`) REFERENCES `markov_transcripts` (`ID`),
	ADD CONSTRAINT `FK_PREV` FOREIGN KEY (`Prev`) REFERENCES `markov_transcripts` (`ID`),
	ADD CONSTRAINT `FK_PREV2` FOREIGN KEY (`Prev2`) REFERENCES `markov_transcripts` (`ID`)//
  
ALTER TABLE `markov_transcripts`
	ADD CONSTRAINT `FK_TRANSCRIPTS_SPEAKER` FOREIGN KEY (`SpeakerID`) REFERENCES `markov_transcripts_speaker` (`ID`)//

ALTER TABLE `markov_transcripts_speaker`
	ALTER `Speaker` DROP DEFAULT//
ALTER TABLE `markov_transcripts_speaker`
	MODIFY COLUMN `Speaker` VARCHAR(128) NOT NULL AFTER `ID`//
  
DROP FUNCTION IF EXISTS `AddMarkov`//
CREATE FUNCTION `AddMarkov`(
	`_prev` BIGINT,
	`_prev2` BIGINT,
	`_speaker` VARCHAR(128),
	`_phrase` VARCHAR(64)
)
RETURNS bigint(20)
LANGUAGE SQL
DETERMINISTIC
MODIFIES SQL DATA
SQL SECURITY DEFINER
COMMENT ''
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

DROP PROCEDURE IF EXISTS `ResetMarkov`//
CREATE PROCEDURE `ResetMarkov`()
LANGUAGE SQL
NOT DETERMINISTIC
MODIFIES SQL DATA
SQL SECURITY DEFINER
COMMENT ''
BEGIN

SET foreign_key_checks = 0;
TRUNCATE markov_transcripts;
TRUNCATE markov_transcripts_speaker;
TRUNCATE markov_transcripts_map;
SET foreign_key_checks = 1;
ALTER TABLE `markov_transcripts` AUTO_INCREMENT=0;
ALTER TABLE `markov_transcripts_speaker` AUTO_INCREMENT=0;
INSERT INTO markov_transcripts_speaker (Speaker)
VALUES ('');
INSERT INTO markov_transcripts (ID, SpeakerID, Phrase)
VALUES (0, 1, '');
UPDATE markov_transcripts SET ID = 0 WHERE ID = 1;

END//


DROP FUNCTION IF EXISTS `GetMarkovLine`//
CREATE FUNCTION `GetMarkovLine`(`_prev` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
    READS SQL DATA
BEGIN

DECLARE line VARCHAR(1024) DEFAULT '|';
SET @prev = _prev;
IF NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev) THEN
	RETURN '|';
END IF;

SET @prev = GetMarkov(@prev);
SET @actionid = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = '');
SET @speakerid = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @prev);
SET @speaker = (SELECT Speaker FROM markov_transcripts_speaker WHERE ID = @speakerid);
SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
SET @max = 0;

IF @speaker = '' THEN
	IF @phrase = '' THEN RETURN CONCAT('|', @prev); END IF;
	SET line = CONCAT('[', @phrase);
ELSE
	SET line = CONCAT('**', @speaker, ':** ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
END IF;

markov_loop: LOOP
	IF @max > 300 OR NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev) THEN LEAVE markov_loop; END IF;
	SET @max = @max + 1;
	SET @capitalize = @phrase = '.' OR @phrase = '!' OR @phrase = '?';
	
	SET @next = GetMarkov(@prev);
	SET @ns = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
	IF @speakerid != @ns THEN LEAVE markov_loop; END IF;
	SET @prev = @next;
	
	SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
	IF @phrase = '.' OR @phrase = '!' OR @phrase = '?' OR @phrase = ',' THEN
		SET line = CONCAT(line, @phrase);
	ELSE
		IF @capitalize THEN
			SET line = CONCAT(line, ' ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
		ELSE
			SET line = CONCAT(line, ' ', @phrase);
		END IF;
	END IF;
END LOOP markov_loop;

IF @speaker = '' THEN
	SET line = CONCAT(line, ']');
END IF;
RETURN CONCAT(line, '|', @prev);

END//

DROP FUNCTION IF EXISTS `GetMarkovLine2`//
CREATE FUNCTION `GetMarkovLine2`(`_prev` BIGINT, `_prev2` BIGINT) RETURNS varchar(1024) CHARSET utf8mb4
    READS SQL DATA
BEGIN

DECLARE line VARCHAR(1024) DEFAULT '|';
SET @prev = _prev;
SET @prev2 = _prev2;
IF NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev AND Prev2 = @prev2) THEN
	RETURN '|';
END IF;

SET @next = GetMarkov2(@prev, @prev2);
SET @actionid = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = '');
SET @speakerid = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
SET @speaker = (SELECT Speaker FROM markov_transcripts_speaker WHERE ID = @speakerid);
SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @next);
SET @max = 0;
SET @prev2 = @prev;
SET @prev = @next;

IF @speaker = '' THEN
	IF @phrase = '' THEN RETURN CONCAT('|', @prev, '|', @prev2); END IF;
	SET line = CONCAT('[', @phrase);
ELSE
	SET line = CONCAT('**', @speaker, ':** ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
END IF;

markov_loop: LOOP
	IF @max > 300 OR NOT EXISTS (SELECT 1 FROM markov_transcripts_map WHERE Prev = @prev AND Prev2 = @prev2) THEN LEAVE markov_loop; END IF;
	SET @max = @max + 1;
	SET @capitalize = @phrase = '.' OR @phrase = '!' OR @phrase = '?';
	
	SET @next = GetMarkov2(@prev, @prev2);
	SET @ns = (SELECT SpeakerID FROM markov_transcripts WHERE ID = @next);
	IF @speakerid != @ns THEN LEAVE markov_loop; END IF;
  SET @prev2 = @prev;
	SET @prev = @next;
	
	SET @phrase = (SELECT Phrase FROM markov_transcripts WHERE ID = @prev);
	IF @phrase = '.' OR @phrase = '!' OR @phrase = '?' OR @phrase = ',' THEN
		SET line = CONCAT(line, @phrase);
	ELSE
		IF @capitalize THEN
			SET line = CONCAT(line, ' ', CONCAT(UCASE(LEFT(@phrase, 1)), SUBSTRING(@phrase, 2)));
		ELSE
			SET line = CONCAT(line, ' ', @phrase);
		END IF;
	END IF;
END LOOP markov_loop;

IF @speaker = '' THEN
	SET line = CONCAT(line, ']');
END IF;
RETURN CONCAT(line, '|', @prev, '|', @prev2);

END//