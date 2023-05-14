ALTER TABLE `rio_services`.`stubs`
ADD COLUMN `tag` VARCHAR(127) DEFAULT '',
ADD INDEX `idx_tag` (`tag`),
ADD INDEX `idx_namespace` (`namespace`);

ALTER TABLE `rio_services`.`incoming_requests`
ADD COLUMN `tag` VARCHAR(127) DEFAULT '',
ADD INDEX `idx_tag` (`tag`),
ADD INDEX `idx_namespace` (`namespace`);



