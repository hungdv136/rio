-- -----------------------------------------------------
-- Table `rio_services`.`protos`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `rio_services`.`protos` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL DEFAULT '',
  `file_id` VARCHAR(63) NOT NULL DEFAULT '',
  `methods` JSON NULL,
  `types` JSON NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_updated_at` (`updated_at`))
ENGINE = InnoDB;

ALTER TABLE `rio_services`.`stubs`
ADD COLUMN `protocol` VARCHAR(31) DEFAULT 'http',
ADD INDEX `idx_protocol` (`protocol`),
ADD INDEX `idx_updated_at` (`updated_at`);