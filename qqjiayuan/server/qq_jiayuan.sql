/*
 Navicat Premium Data Transfer

 Source Server         : localhost
 Source Server Type    : MySQL
 Source Server Version : 80405 (8.4.5)
 Source Host           : localhost:3306
 Source Schema         : qq_jiayuan

 Target Server Type    : MySQL
 Target Server Version : 80405 (8.4.5)
 File Encoding         : 65001

 Date: 07/09/2026 16:10:11
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for albums
-- ----------------------------
DROP TABLE IF EXISTS `albums`;
CREATE TABLE `albums`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `cover` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `count` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_albums_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of albums
-- ----------------------------

-- ----------------------------
-- Table structure for announcements
-- ----------------------------
DROP TABLE IF EXISTS `announcements`;
CREATE TABLE `announcements`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `type` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of announcements
-- ----------------------------
INSERT INTO `announcements` VALUES (1, 'notice', '欢迎来到家园社区', '在这里，玩家可以随时随地的和好友进行互动，一起玩游戏。社区常年招募管理员与版主，有意者到客服中心申请。', 1, '2026-08-30 10:07:01.237', '2026-08-30 14:46:24.187');
INSERT INTO `announcements` VALUES (2, 'broadcast', '行百里者，半于九十', '小Q广播：走一百里路，走了九十里才算走了一半。越接近成功越要认真对待！', 1, '2026-08-30 10:07:01.237', '2026-08-30 10:07:01.237');
INSERT INTO `announcements` VALUES (3, 'activity', '五一活动之《歌王就是你》第二届举办帖', '活动时间：即日起至月底。参与方式：在休闲灌水板块发布你的拿手歌曲翻唱帖，回帖数前三名获得社区勋章与金币奖励！', 1, '2026-08-30 10:07:01.237', '2026-08-30 10:07:01.237');

-- ----------------------------
-- Table structure for articles
-- ----------------------------
DROP TABLE IF EXISTS `articles`;
CREATE TABLE `articles`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_articles_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of articles
-- ----------------------------

-- ----------------------------
-- Table structure for badges
-- ----------------------------
DROP TABLE IF EXISTS `badges`;
CREATE TABLE `badges`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `icon` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 23 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of badges
-- ----------------------------
INSERT INTO `badges` VALUES (12, '公坛协管员', '706.jpg', '社区职务徽章', 1);
INSERT INTO `badges` VALUES (13, '公坛管理', '704.gif', '公坛管理组', 1);
INSERT INTO `badges` VALUES (14, '客服专员', '3.gif', '客服团专属', 1);
INSERT INTO `badges` VALUES (15, '社区传媒', '501.gif', '时报记者专属', 1);
INSERT INTO `badges` VALUES (16, '老友归来', '803.gif', '回归纪念', 1);
INSERT INTO `badges` VALUES (17, '原创写手', '804.gif', '文学贡献', 1);
INSERT INTO `badges` VALUES (18, '贵族一级', '103.gif', '贵族身份', 1);
INSERT INTO `badges` VALUES (19, '贵族二级', '15.gif', '贵族身份', 1);
INSERT INTO `badges` VALUES (20, '尊上', '903.gif', '传说中的马甲', 1);
INSERT INTO `badges` VALUES (21, '表情大师', '45.gif', '斗图冠军', 1);

-- ----------------------------
-- Table structure for bank_accounts
-- ----------------------------
DROP TABLE IF EXISTS `bank_accounts`;
CREATE TABLE `bank_accounts`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `balance` bigint NULL DEFAULT 0,
  `last_interest_at` datetime(3) NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_bank_accounts_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of bank_accounts
-- ----------------------------
INSERT INTO `bank_accounts` VALUES (1, 10001, 100, NULL, '2026-09-04 09:51:25.093', '2026-09-04 09:51:25.103');

-- ----------------------------
-- Table structure for boards
-- ----------------------------
DROP TABLE IF EXISTS `boards`;
CREATE TABLE `boards`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `parent_id` bigint UNSIGNED NULL DEFAULT 0,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `description` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `sort` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  `thread_count` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_boards_parent_id`(`parent_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 35 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of boards
-- ----------------------------
INSERT INTO `boards` VALUES (1, 0, '公共论坛', '最大的公共讨论区，畅所欲言', 1, 1, 0, '2026-08-30 10:07:01.139', '2026-08-30 10:07:01.139');
INSERT INTO `boards` VALUES (2, 0, '家族大厅', '家族组建、申请、风云榜', 0, 1, 0, '2026-08-30 10:07:01.158', '2026-08-30 14:46:23.883');
INSERT INTO `boards` VALUES (3, 0, '同城客栈', '累了吗？来同城客栈透个气吧', 3, 1, 0, '2026-08-30 10:07:01.162', '2026-08-30 10:07:01.162');
INSERT INTO `boards` VALUES (4, 0, '社区服务', '客服、建议、公示', 4, 1, 0, '2026-08-30 10:07:01.167', '2026-08-30 10:07:01.167');
INSERT INTO `boards` VALUES (5, 1, '休闲灌水', '没事灌灌水，聊聊日常', 2, 1, 9, '2026-08-30 10:07:01.172', '2026-09-07 15:38:45.177');
INSERT INTO `boards` VALUES (6, 1, '时尚美眉', '美眉们的时尚领地', 1, 1, 1, '2026-08-30 10:07:01.177', '2026-09-07 15:38:45.167');
INSERT INTO `boards` VALUES (7, 1, '新人求助', '新手报到、不懂就问', 4, 1, 5, '2026-08-30 10:07:01.181', '2026-09-07 15:38:45.145');
INSERT INTO `boards` VALUES (8, 1, '情感天地', '缘分、心情日记、鹊桥相会', 5, 1, 3, '2026-08-30 10:07:01.186', '2026-09-07 15:38:45.151');
INSERT INTO `boards` VALUES (9, 1, '数码动漫', '玩转手机、游戏狂潮、动漫天地', 6, 1, 3, '2026-08-30 10:07:01.190', '2026-09-07 15:38:45.158');
INSERT INTO `boards` VALUES (10, 2, '家族申请', '申请建立你的家族', 0, 1, 2, '2026-08-30 10:07:01.194', '2026-08-30 10:07:01.194');
INSERT INTO `boards` VALUES (11, 2, '家族风云', '家族动态与风云榜', 0, 1, 2, '2026-08-30 10:07:01.199', '2026-08-30 10:07:01.199');
INSERT INTO `boards` VALUES (12, 3, '福建', '福建的老乡看过来', 0, 1, 1, '2026-08-30 10:07:01.204', '2026-08-30 10:07:01.204');
INSERT INTO `boards` VALUES (13, 3, '北京', '北京的老乡看过来', 0, 1, 1, '2026-08-30 10:07:01.208', '2026-08-30 10:07:01.208');
INSERT INTO `boards` VALUES (14, 3, '广东', '广东的老乡看过来', 0, 1, 0, '2026-08-30 10:07:01.213', '2026-08-30 10:07:01.213');
INSERT INTO `boards` VALUES (15, 3, '共建同城', '你的城市还没有？来这里申请', 0, 1, 0, '2026-08-30 10:07:01.218', '2026-08-30 10:07:01.218');
INSERT INTO `boards` VALUES (16, 4, '客服中心', '投诉、找回资料、社区事务', 0, 1, 0, '2026-08-30 10:07:01.222', '2026-08-30 10:07:01.222');
INSERT INTO `boards` VALUES (17, 4, '意见建议', '为家园建设献言献策', 0, 1, 1, '2026-08-30 10:07:01.226', '2026-08-30 10:07:01.226');
INSERT INTO `boards` VALUES (18, 4, '社区公示', '封禁公示、人事任免', 0, 1, 0, '2026-08-30 10:07:01.230', '2026-08-30 10:07:01.230');
INSERT INTO `boards` VALUES (19, 0, '游戏论坛', '各游戏交流区：攻略、晒图、组队、交易', 5, 1, 0, '2026-08-30 12:33:43.971', '2026-08-30 12:33:43.971');
INSERT INTO `boards` VALUES (20, 19, '幻想西游', '经典wap游戏，古典神话网游，再梦西游', 0, 1, 1, '2026-08-30 12:33:43.978', '2026-08-30 12:33:43.978');
INSERT INTO `boards` VALUES (21, 19, '永恒修仙', '经典wap游戏，永恒修仙，欢迎体验', 0, 1, 0, '2026-08-30 12:33:43.984', '2026-08-30 12:33:43.984');
INSERT INTO `boards` VALUES (22, 19, '魔法花园', '花的世界，花的海洋，花的物语', 0, 1, 1, '2026-08-30 12:33:43.990', '2026-08-30 12:33:43.990');
INSERT INTO `boards` VALUES (23, 19, '婚礼殿堂', '闯荡社区快来：婚姻礼堂，寻找爱的另一半！', 0, 1, 0, '2026-08-30 12:33:43.995', '2026-08-30 12:33:43.995');
INSERT INTO `boards` VALUES (24, 19, '开心农场', '开心农场，播种开心，收获快乐', 0, 1, 0, '2026-08-30 12:33:44.000', '2026-08-30 12:33:44.000');
INSERT INTO `boards` VALUES (25, 19, '狂抢车位', '停放车辆，展现身价，乐趣无穷', 0, 1, 1, '2026-08-30 12:33:44.005', '2026-08-30 12:33:44.005');
INSERT INTO `boards` VALUES (26, 19, '精武堂', '江湖格斗，残酷厮杀，随死即生', 0, 1, 1, '2026-08-30 12:33:44.010', '2026-08-30 12:33:44.010');
INSERT INTO `boards` VALUES (27, 19, '家园宠物', '家园宠物，内测中', 0, 1, 0, '2026-08-30 12:33:44.015', '2026-08-30 12:33:44.015');
INSERT INTO `boards` VALUES (28, 19, '水果乐园', '轻松娱乐，点缀生活，水果乐园', 0, 1, 0, '2026-08-30 12:33:44.020', '2026-08-30 12:33:44.020');
INSERT INTO `boards` VALUES (29, 19, '全民猎马', '周二四六，包你赢够，尽在猎马', 0, 1, 0, '2026-08-30 12:33:44.025', '2026-08-30 12:33:44.025');
INSERT INTO `boards` VALUES (30, 19, '家园股市', '3GQQ家园股市，一夜成名，瞬间暴富', 0, 1, 0, '2026-08-30 12:33:44.031', '2026-08-30 12:33:44.031');
INSERT INTO `boards` VALUES (31, 19, '大话吹牛', '大话吹牛，打打闹闹，更是乐哉', 0, 1, 0, '2026-08-30 12:33:44.036', '2026-08-30 12:33:44.036');
INSERT INTO `boards` VALUES (32, 1, '公坛事务', '公坛区域管理：管理须知、花名册、公示', 3, 1, 4, '2026-08-30 13:44:09.983', '2026-09-07 15:38:45.137');
INSERT INTO `boards` VALUES (33, 2, '家族大看台', '家族活动、家族大事一览', 0, 1, 0, '2026-09-05 10:15:54.878', '2026-09-05 10:15:54.878');
INSERT INTO `boards` VALUES (34, 2, '家族·清风明月', '清风明月 家族论坛', 0, 1, 0, '2026-09-05 10:16:14.875', '2026-09-05 10:16:14.875');

-- ----------------------------
-- Table structure for books
-- ----------------------------
DROP TABLE IF EXISTS `books`;
CREATE TABLE `books`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `author` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `category` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `intro` varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT '连载',
  `recommend` bigint NULL DEFAULT 0,
  `new_book` bigint NULL DEFAULT 0,
  `rating` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT '★★★★★',
  `views` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 9 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of books
-- ----------------------------
INSERT INTO `books` VALUES (1, '剑影江湖', '家园侠客', '武侠', '乱世出英雄，一柄长剑闯天涯。家国恩怨，儿女情长。', '连载', 1, 0, '★★★★★', 3, '2026-09-04 11:05:22.401', '2026-09-04 11:10:31.859');
INSERT INTO `books` VALUES (2, '山河故人', '云深不知', '武侠', '倦鸟归林，故人相逢。旧时刀剑，今朝煮茶。', '连载', 0, 0, '★★★★', 0, '2026-09-04 11:05:22.405', '2026-09-04 11:05:22.405');
INSERT INTO `books` VALUES (3, '仲夏绮梦', '木槿昔年', '言情', '那年仲夏，蝉鸣与少年，都是青春最美的模样。', '完结', 1, 0, '★★★★★', 0, '2026-09-04 11:05:22.410', '2026-09-04 11:05:22.410');
INSERT INTO `books` VALUES (4, '碎碎念', '文墨', '都市', '都市里的烟火气，柴米油盐也动人。', '连载', 0, 0, '★★★', 0, '2026-09-04 11:05:22.415', '2026-09-04 11:05:22.415');
INSERT INTO `books` VALUES (5, '盗墓', '夜行人', '灵异', '地下的秘密，随着灯火一盏盏熄灭。', '连载', 1, 0, '★★★★', 0, '2026-09-04 11:05:22.419', '2026-09-04 11:05:22.419');
INSERT INTO `books` VALUES (6, '忘忧奶茶店', '小甜', '都市', '一杯奶茶，换你一个故事。', '完结', 0, 1, '★★★★', 0, '2026-09-04 11:05:22.423', '2026-09-04 11:05:22.423');
INSERT INTO `books` VALUES (7, '六零：城里小白菜回乡当团宠', '阿园', '言情', '穿越六零，从城里小白菜到乡间团宠。', '连载', 1, 1, '★★★★★', 0, '2026-09-04 11:05:22.428', '2026-09-04 11:05:22.428');
INSERT INTO `books` VALUES (8, '大汉宏图', '子夜', '武侠', '铁血王朝，宏图霸业，一将功成万骨枯。', '连载', 0, 1, '★★★★', 0, '2026-09-04 11:05:22.432', '2026-09-04 11:05:22.432');

-- ----------------------------
-- Table structure for chat_messages
-- ----------------------------
DROP TABLE IF EXISTS `chat_messages`;
CREATE TABLE `chat_messages`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `family_id` bigint UNSIGNED NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_chat_messages_user_id`(`user_id` ASC) USING BTREE,
  INDEX `idx_chat_messages_family_id`(`family_id` ASC) USING BTREE,
  CONSTRAINT `fk_chat_messages_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of chat_messages
-- ----------------------------
INSERT INTO `chat_messages` VALUES (2, 10000, '大家晚上好，聊天室开张啦！', '2026-08-30 10:32:39.872', 0);
INSERT INTO `chat_messages` VALUES (3, 10007, '大家好', '2026-08-30 10:37:30.115', 0);

-- ----------------------------
-- Table structure for donations
-- ----------------------------
DROP TABLE IF EXISTS `donations`;
CREATE TABLE `donations`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `amount` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_donations_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of donations
-- ----------------------------
INSERT INTO `donations` VALUES (1, 10001, 10, '2026-09-04 16:19:58.303');
INSERT INTO `donations` VALUES (2, 10001, 10, '2026-09-05 16:30:14.441');

-- ----------------------------
-- Table structure for families
-- ----------------------------
DROP TABLE IF EXISTS `families`;
CREATE TABLE `families`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `slogan` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `description` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `announcement` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `owner_id` bigint UNSIGNED NULL DEFAULT NULL,
  `tree_level` bigint NULL DEFAULT 1,
  `tree_exp` bigint NULL DEFAULT 0,
  `battle_score` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `category` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `is_feature` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_families_owner_id`(`owner_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of families
-- ----------------------------
INSERT INTO `families` VALUES (1, '清风明月', '轻风徐来，明月入怀', '以文会友，共话家常。', '欢迎回家，常来常往。', 10001, 4, 311, 385, 1, '2026-09-04 09:59:54.303', '2026-09-07 15:38:45.222', '舞文弄墨', 1);
INSERT INTO `families` VALUES (2, '与世无争', '与世无争，不问西东', '恬淡生活，其乐融融。', '兄弟姐妹们常回家看看。', 10002, 5, 151, 260, 1, '2026-09-04 09:59:54.324', '2026-09-07 15:38:45.228', '情感男女', 1);
INSERT INTO `families` VALUES (3, '断念阁', '聚是一团火，散是满天星', '以舞会友，以歌传情。', '新老朋友皆可入阁。', 35797804, 2, 181, 195, 1, '2026-09-04 09:59:54.338', '2026-09-04 10:12:53.816', '青春校园', 0);

-- ----------------------------
-- Table structure for family_activities
-- ----------------------------
DROP TABLE IF EXISTS `family_activities`;
CREATE TABLE `family_activities`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `family_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(160) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_family_activities_family_id`(`family_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 18 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of family_activities
-- ----------------------------
INSERT INTO `family_activities` VALUES (1, 1, 10001, '在家族签到', '2026-09-04 10:12:22.135');
INSERT INTO `family_activities` VALUES (2, 1, 10001, '抚摸/拥抱了守护树', '2026-09-04 10:12:22.140');
INSERT INTO `family_activities` VALUES (3, 1, 10001, '参加了家族乐斗，战胜了对手', '2026-09-04 10:12:22.144');
INSERT INTO `family_activities` VALUES (4, 1, 10001, '分享了家族公告', '2026-09-04 10:12:22.149');
INSERT INTO `family_activities` VALUES (5, 2, 10002, '在家族签到', '2026-09-04 10:12:22.155');
INSERT INTO `family_activities` VALUES (6, 1, 10007, '加入了家族《清风明月》', '2026-09-04 22:20:48.488');
INSERT INTO `family_activities` VALUES (7, 1, 10007, '在家族签到', '2026-09-05 09:14:16.961');
INSERT INTO `family_activities` VALUES (8, 1, 10007, '抚摸/拥抱了守护树', '2026-09-05 09:14:17.809');
INSERT INTO `family_activities` VALUES (9, 1, 10007, '参加家族乐斗，惜败于对手《断念阁》', '2026-09-05 09:14:18.400');
INSERT INTO `family_activities` VALUES (10, 1, 10007, '退出了家族', '2026-09-05 09:14:18.865');
INSERT INTO `family_activities` VALUES (11, 1, 10007, '加入了家族《清风明月》', '2026-09-05 09:14:21.764');
INSERT INTO `family_activities` VALUES (12, 1, 10001, '更新了家族公告', '2026-09-05 14:57:25.184');
INSERT INTO `family_activities` VALUES (13, 1, 10001, '参加家族乐斗，惜败于对手《断念阁》', '2026-09-05 14:57:28.685');
INSERT INTO `family_activities` VALUES (14, 1, 10001, '参加家族乐斗，战胜了对手《与世无争》', '2026-09-05 14:57:30.555');
INSERT INTO `family_activities` VALUES (15, 1, 10001, '参加家族乐斗，惜败于对手《与世无争》', '2026-09-05 14:57:31.845');
INSERT INTO `family_activities` VALUES (16, 1, 10001, '参加家族乐斗，战胜了对手《与世无争》', '2026-09-05 14:57:32.991');
INSERT INTO `family_activities` VALUES (17, 1, 10001, '参加家族乐斗，惜败于对手《与世无争》', '2026-09-05 15:02:47.197');

-- ----------------------------
-- Table structure for family_members
-- ----------------------------
DROP TABLE IF EXISTS `family_members`;
CREATE TABLE `family_members`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `family_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `role` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT 'member',
  `exp` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_fam_user`(`user_id` ASC) USING BTREE,
  INDEX `idx_family_members_family_id`(`family_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 10 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of family_members
-- ----------------------------
INSERT INTO `family_members` VALUES (1, 1, 10001, 'owner', 340, '2026-09-04 09:59:54.308');
INSERT INTO `family_members` VALUES (2, 1, 10003, 'member', 51, '2026-09-04 09:59:54.314');
INSERT INTO `family_members` VALUES (3, 1, 10004, 'member', 71, '2026-09-04 09:59:54.320');
INSERT INTO `family_members` VALUES (4, 2, 10002, 'owner', 320, '2026-09-04 09:59:54.329');
INSERT INTO `family_members` VALUES (5, 2, 10005, 'member', 31, '2026-09-04 09:59:54.333');
INSERT INTO `family_members` VALUES (6, 3, 35797804, 'owner', 320, '2026-09-04 09:59:54.342');
INSERT INTO `family_members` VALUES (9, 1, 10007, 'member', 0, '2026-09-05 09:14:21.760');

-- ----------------------------
-- Table structure for family_sign_ins
-- ----------------------------
DROP TABLE IF EXISTS `family_sign_ins`;
CREATE TABLE `family_sign_ins`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `family_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sign_date` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `type` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT 'sign',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_family_sign_ins_family_id`(`family_id` ASC) USING BTREE,
  INDEX `idx_family_sign_ins_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 7 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of family_sign_ins
-- ----------------------------
INSERT INTO `family_sign_ins` VALUES (1, 1, 10001, '2026-09-04', '2026-09-04 10:00:09.218', 'sign');
INSERT INTO `family_sign_ins` VALUES (2, 1, 10001, '2026-09-04', '2026-09-04 10:01:45.539', 'tree');
INSERT INTO `family_sign_ins` VALUES (3, 3, 10007, '2026-09-04', '2026-09-04 10:02:34.848', 'sign');
INSERT INTO `family_sign_ins` VALUES (4, 3, 10007, '2026-09-04', '2026-09-04 10:02:36.908', 'tree');
INSERT INTO `family_sign_ins` VALUES (5, 1, 10007, '2026-09-05', '2026-09-05 09:14:16.943', 'sign');
INSERT INTO `family_sign_ins` VALUES (6, 1, 10007, '2026-09-05', '2026-09-05 09:14:17.790', 'tree');

-- ----------------------------
-- Table structure for friend_group_items
-- ----------------------------
DROP TABLE IF EXISTS `friend_group_items`;
CREATE TABLE `friend_group_items`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `friend_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_friend_group_items_group_id`(`group_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of friend_group_items
-- ----------------------------
INSERT INTO `friend_group_items` VALUES (1, 1, 10001, 10001, '2026-09-04 11:31:02.256');
INSERT INTO `friend_group_items` VALUES (2, 2, 10007, 10007, '2026-09-04 21:10:53.385');

-- ----------------------------
-- Table structure for friend_groups
-- ----------------------------
DROP TABLE IF EXISTS `friend_groups`;
CREATE TABLE `friend_groups`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_friend_groups_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of friend_groups
-- ----------------------------
INSERT INTO `friend_groups` VALUES (1, 10001, '??', '2026-09-04 11:31:02.198', '2026-09-04 11:31:02.198');
INSERT INTO `friend_groups` VALUES (2, 10007, '999', '2026-09-04 11:34:32.640', '2026-09-04 11:34:32.640');

-- ----------------------------
-- Table structure for friendships
-- ----------------------------
DROP TABLE IF EXISTS `friendships`;
CREATE TABLE `friendships`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `friend_id` bigint UNSIGNED NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_friendships_user_id`(`user_id` ASC) USING BTREE,
  INDEX `idx_friendships_friend_id`(`friend_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 6 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of friendships
-- ----------------------------
INSERT INTO `friendships` VALUES (1, 10001, 10002, 1, '2026-08-30 10:08:24.511', '2026-08-30 10:08:25.004');
INSERT INTO `friendships` VALUES (2, 10007, 10000, 1, '2026-08-30 10:37:33.619', '2026-08-30 19:43:17.527');
INSERT INTO `friendships` VALUES (3, 10007, 10004, 0, '2026-08-30 10:38:00.946', '2026-08-30 10:38:00.946');
INSERT INTO `friendships` VALUES (4, 10007, 35797804, 0, '2026-08-30 14:01:04.917', '2026-08-30 14:01:04.917');
INSERT INTO `friendships` VALUES (5, 10000, 35797804, 0, '2026-08-30 14:17:03.963', '2026-08-30 14:17:03.963');

-- ----------------------------
-- Table structure for games
-- ----------------------------
DROP TABLE IF EXISTS `games`;
CREATE TABLE `games`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `category` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `logo` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `stars` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `desc` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `board_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sort` bigint NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `url` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 14 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of games
-- ----------------------------
INSERT INTO `games` VALUES (1, '幻想西游', 'net', '', '★★★★★', '经典wap游戏，古典神话网游，再梦西游。持神兵利器，降五爪金龙，携爱行走西游', 20, 1, 1, NULL);
INSERT INTO `games` VALUES (2, '永恒修仙', 'net', 'logo.jpg', '★★★★★', '经典wap游戏，永恒修仙。欢迎体验', 21, 2, 1, NULL);
INSERT INTO `games` VALUES (3, '魔法花园', 'com', 'mofahuayuan.gif', '★★★★★', '花的世界，花的海洋，花的物语', 22, 1, 1, NULL);
INSERT INTO `games` VALUES (4, '婚礼殿堂', 'com', 'hunli2.jpg', '★★★★★', '闯荡社区快来: 婚姻礼堂 寻找爱的另一半！', 23, 2, 1, NULL);
INSERT INTO `games` VALUES (5, '开心农场', 'com', 'kaixinnongchang.gif', '★★★★☆', '开心农场，播种开心，收获快乐', 24, 3, 1, NULL);
INSERT INTO `games` VALUES (6, '狂抢车位', 'com', 'kuangqiangchewei.gif', '★★★☆☆', '停放车辆，展现身价，乐趣无穷', 25, 4, 1, NULL);
INSERT INTO `games` VALUES (7, '精武堂', 'com', 'jwt.png', '★★★★★', '江湖格斗，残酷厮杀，随死即生', 26, 5, 1, NULL);
INSERT INTO `games` VALUES (8, '家园宠物', 'com', 'cwlogo.gif', '★★', '家园宠物，内测中', 27, 6, 1, NULL);
INSERT INTO `games` VALUES (9, '水果乐园', 'com', 'shuiguoleyuan.gif', '★★☆☆☆', '轻松娱乐，点缀生活，水果乐园', 28, 7, 1, NULL);
INSERT INTO `games` VALUES (10, '全民猎马', 'com', 'quanminliema.gif', '★★★★☆', '周二四六，包你赢够，尽在猎马', 29, 8, 1, NULL);
INSERT INTO `games` VALUES (11, '家园股市', 'com', 'jiayuangushi.gif', '★☆☆☆☆', '3GQQ家园股市，一夜成名，瞬间暴富', 30, 9, 1, NULL);
INSERT INTO `games` VALUES (12, '大话吹牛', 'com', 'dahuachuiniu.gif', '★★★☆☆', '大话吹牛，打打闹闹，更是乐哉', 31, 10, 1, NULL);

-- ----------------------------
-- Table structure for garden_activities
-- ----------------------------
DROP TABLE IF EXISTS `garden_activities`;
CREATE TABLE `garden_activities`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `desc` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `needs` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `reward` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 6 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_activities
-- ----------------------------
INSERT INTO `garden_activities` VALUES (1, '春天的爱恋', '春天来了，魔法花园里的花儿沐浴着温馨的春风和绵绵的春雨含苞待放着，想要在春天绽放自己最美的身影。花仙子陶醉在浓浓的春意中，撒下了许多象征着爱情的朝暮盈霄花种子，快去寻找吧！', 1, '2026-09-04 13:19:15.572', '2026-09-07 15:38:45.237', '[{\"flower\":\"红玫瑰\",\"n\":6},{\"flower\":\"红桃花\",\"n\":6},{\"flower\":\"红色勿忘我\",\"n\":6},{\"flower\":\"红色烈焰焚情\",\"n\":6}]', '朝暮盈霄花');
INSERT INTO `garden_activities` VALUES (2, '小魔女的烦恼', '小魔女：“每次聚会都要hold住全场，不够鲜花装扮自己怎么办呀！谁能送我一些鲜花，我会给TA丰厚的回报哟！”', 1, '2026-09-04 13:19:15.580', '2026-09-07 15:38:45.245', '[{\"flower\":\"红色菊花\",\"n\":3},{\"flower\":\"红色野花\",\"n\":3},{\"flower\":\"红桃花\",\"n\":3},{\"flower\":\"红兰花\",\"n\":3}]', '夜魔南瓜花');
INSERT INTO `garden_activities` VALUES (3, '花仙子的新房子', '花仙子：“555，我的花房有些时候没有修整了，天气开始转凉，我都被冻感冒几次了，我急需一些花来重新补整我的花房，请你帮我去采些丁香/樱花/野花/梅花/兰花，我会拿最新出的步步高升花种回报你哦！”', 1, '2026-09-04 13:19:15.590', '2026-09-07 15:38:45.253', '[{\"flower\":\"红丁香\",\"n\":1},{\"flower\":\"红樱花\",\"n\":1},{\"flower\":\"红色野花\",\"n\":1},{\"flower\":\"红梅花\",\"n\":1},{\"flower\":\"红兰花\",\"n\":1}]', '步步高升');
INSERT INTO `garden_activities` VALUES (4, '寻找遗失的碎片', '想要开启花园的精灵花册，需要集齐对应的珍惜碎片，快来用你种出来的鲜花和我兑换！', 1, '2026-09-04 13:19:15.594', '2026-09-07 15:38:45.260', '[{\"flower\":\"红玫瑰\",\"n\":33},{\"flower\":\"黄玫瑰\",\"n\":33},{\"flower\":\"白玫瑰\",\"n\":33},{\"flower\":\"粉玫瑰\",\"n\":33},{\"flower\":\"银玫瑰\",\"n\":3},{\"flower\":\"金玫瑰\",\"n\":1}]', '玫瑰金碎片');

-- ----------------------------
-- Table structure for garden_bags
-- ----------------------------
DROP TABLE IF EXISTS `garden_bags`;
CREATE TABLE `garden_bags`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `seed_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `amount` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_bags_user_id`(`user_id` ASC) USING BTREE,
  INDEX `idx_garden_bags_seed_id`(`seed_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 13 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_bags
-- ----------------------------
INSERT INTO `garden_bags` VALUES (2, 10001, 2, '玫瑰花', 1);
INSERT INTO `garden_bags` VALUES (3, 10000, 14, '银色烈焰焚情', 1);
INSERT INTO `garden_bags` VALUES (5, 10000, 1, '向日葵', 2);
INSERT INTO `garden_bags` VALUES (6, 10000, 8, '樱花', 2);
INSERT INTO `garden_bags` VALUES (7, 10000, 18, '染色药水', 2);
INSERT INTO `garden_bags` VALUES (8, 10000, 3, '郁金香', 1);
INSERT INTO `garden_bags` VALUES (12, 10007, 1, '向日葵', 10);

-- ----------------------------
-- Table structure for garden_bottles
-- ----------------------------
DROP TABLE IF EXISTS `garden_bottles`;
CREATE TABLE `garden_bottles`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `flower` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `count` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_gb`(`user_id` ASC, `flower` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_bottles
-- ----------------------------
INSERT INTO `garden_bottles` VALUES (1, 10001, '红玫瑰', 1);
INSERT INTO `garden_bottles` VALUES (2, 10007, '???', 3);

-- ----------------------------
-- Table structure for garden_elf_logs
-- ----------------------------
DROP TABLE IF EXISTS `garden_elf_logs`;
CREATE TABLE `garden_elf_logs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `elf_id` bigint UNSIGNED NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_gel`(`user_id` ASC, `elf_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_elf_logs
-- ----------------------------
INSERT INTO `garden_elf_logs` VALUES (1, 10007, 1);
INSERT INTO `garden_elf_logs` VALUES (2, 10007, 2);

-- ----------------------------
-- Table structure for garden_elves
-- ----------------------------
DROP TABLE IF EXISTS `garden_elves`;
CREATE TABLE `garden_elves`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `desc` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `img` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `need_map` bigint NULL DEFAULT 1,
  `sort` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 11 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_elves
-- ----------------------------
INSERT INTO `garden_elves` VALUES (1, '绿芽精灵', '花园的新生，点亮 3 个图谱后觉醒。', 'elf_1.png', 3, 1, 1, '2026-09-06 22:45:47.921', '2026-09-07 15:38:45.285');
INSERT INTO `garden_elves` VALUES (2, '露珠精灵', '清晨的第一滴露水，点亮 6 个图谱后觉醒。', 'elf_2.png', 6, 2, 1, '2026-09-06 22:45:47.926', '2026-09-07 15:38:45.292');
INSERT INTO `garden_elves` VALUES (3, '花粉精灵', '随风飞舞的花粉，点亮 9 个图谱后觉醒。', 'elf_3.png', 9, 3, 1, '2026-09-06 22:45:47.930', '2026-09-07 15:38:45.299');
INSERT INTO `garden_elves` VALUES (4, '花苞精灵', '含苞待放的期待，点亮 12 个图谱后觉醒。', 'elf_4.png', 12, 4, 1, '2026-09-06 22:45:47.934', '2026-09-07 15:38:45.306');
INSERT INTO `garden_elves` VALUES (5, '月光精灵', '月下的银色光辉，点亮 15 个图谱后觉醒。', 'elf_5.png', 15, 5, 1, '2026-09-06 22:45:47.939', '2026-09-07 15:38:45.314');
INSERT INTO `garden_elves` VALUES (6, '彩虹精灵', '七彩的花之桥，点亮 18 个图谱后觉醒。', 'elf_6.png', 18, 6, 1, '2026-09-06 22:45:47.943', '2026-09-07 15:38:45.323');
INSERT INTO `garden_elves` VALUES (7, '星光精灵', '夜空里的花语，点亮 21 个图谱后觉醒。', 'elf_7.png', 21, 7, 1, '2026-09-06 22:45:47.947', '2026-09-07 15:38:45.330');
INSERT INTO `garden_elves` VALUES (8, '晨露精灵', '晨曦中的晶莹，点亮 24 个图谱后觉醒。', 'elf_8.png', 24, 8, 1, '2026-09-06 22:45:47.951', '2026-09-07 15:38:45.337');
INSERT INTO `garden_elves` VALUES (9, '花语精灵', '读懂每一朵花的低语，点亮 27 个图谱后觉醒。', 'elf_9.png', 27, 9, 1, '2026-09-06 22:45:47.957', '2026-09-07 15:38:45.345');
INSERT INTO `garden_elves` VALUES (10, '花园精灵王', '花园世界的守护者，点亮 30 个图谱后降临。', 'elf_10.png', 30, 10, 1, '2026-09-06 22:45:47.961', '2026-09-07 15:38:45.354');

-- ----------------------------
-- Table structure for garden_gifts
-- ----------------------------
DROP TABLE IF EXISTS `garden_gifts`;
CREATE TABLE `garden_gifts`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `from_uid` bigint UNSIGNED NULL DEFAULT NULL,
  `to_uid` bigint UNSIGNED NULL DEFAULT NULL,
  `flower` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `amount` bigint NULL DEFAULT 0,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_gifts_from_uid`(`from_uid` ASC) USING BTREE,
  INDEX `idx_garden_gifts_to_uid`(`to_uid` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_gifts
-- ----------------------------
INSERT INTO `garden_gifts` VALUES (1, 10000, 10001, '红玫瑰', 1, '希望你开心快乐！', '2026-09-06 21:55:23.954');

-- ----------------------------
-- Table structure for garden_land_logs
-- ----------------------------
DROP TABLE IF EXISTS `garden_land_logs`;
CREATE TABLE `garden_land_logs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `land_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_land_logs_land_id`(`land_id` ASC) USING BTREE,
  INDEX `idx_garden_land_logs_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_land_logs
-- ----------------------------
INSERT INTO `garden_land_logs` VALUES (1, 1, 10000);

-- ----------------------------
-- Table structure for garden_map_logs
-- ----------------------------
DROP TABLE IF EXISTS `garden_map_logs`;
CREATE TABLE `garden_map_logs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `map_id` bigint UNSIGNED NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_gml`(`user_id` ASC, `map_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 8 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_map_logs
-- ----------------------------
INSERT INTO `garden_map_logs` VALUES (1, 10000, 3);
INSERT INTO `garden_map_logs` VALUES (3, 10007, 1);
INSERT INTO `garden_map_logs` VALUES (7, 10007, 2);
INSERT INTO `garden_map_logs` VALUES (2, 10007, 3);
INSERT INTO `garden_map_logs` VALUES (5, 10007, 8);
INSERT INTO `garden_map_logs` VALUES (6, 10007, 9);
INSERT INTO `garden_map_logs` VALUES (4, 10007, 19);

-- ----------------------------
-- Table structure for garden_maps
-- ----------------------------
DROP TABLE IF EXISTS `garden_maps`;
CREATE TABLE `garden_maps`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `seed_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `d_type` bigint NULL DEFAULT 0,
  `img` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_maps_seed_id`(`seed_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 34 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_maps
-- ----------------------------
INSERT INTO `garden_maps` VALUES (1, 1, '金色向日葵', 0, '');
INSERT INTO `garden_maps` VALUES (2, 1, '七彩向日葵', 1, '');
INSERT INTO `garden_maps` VALUES (3, 1, '太阳神花', 2, '');
INSERT INTO `garden_maps` VALUES (4, 2, '红玫瑰', 0, '');
INSERT INTO `garden_maps` VALUES (5, 2, '蓝玫瑰', 1, '');
INSERT INTO `garden_maps` VALUES (6, 2, '黑玫瑰', 2, '');
INSERT INTO `garden_maps` VALUES (7, 3, '黄郁金香', 0, '');
INSERT INTO `garden_maps` VALUES (8, 3, '粉郁金香', 1, '');
INSERT INTO `garden_maps` VALUES (9, 3, '黑郁金香', 2, '');
INSERT INTO `garden_maps` VALUES (10, 4, '月光花', 0, '');
INSERT INTO `garden_maps` VALUES (11, 4, '星月花', 1, '');
INSERT INTO `garden_maps` VALUES (12, 4, '幻月花', 2, '');
INSERT INTO `garden_maps` VALUES (13, 5, '白百合', 0, '');
INSERT INTO `garden_maps` VALUES (14, 5, '金百合', 1, '');
INSERT INTO `garden_maps` VALUES (15, 5, '火百合', 2, '');
INSERT INTO `garden_maps` VALUES (16, 6, '粉牡丹', 0, '');
INSERT INTO `garden_maps` VALUES (17, 6, '绿牡丹', 1, '');
INSERT INTO `garden_maps` VALUES (18, 6, '黑牡丹', 2, '');
INSERT INTO `garden_maps` VALUES (19, 7, '蓝色妖姬', 0, '');
INSERT INTO `garden_maps` VALUES (20, 7, '冰蓝妖姬', 1, '');
INSERT INTO `garden_maps` VALUES (21, 7, '魅蓝妖姬', 2, '');
INSERT INTO `garden_maps` VALUES (22, 8, '粉樱花', 0, '');
INSERT INTO `garden_maps` VALUES (23, 8, '垂枝樱', 1, '');
INSERT INTO `garden_maps` VALUES (24, 8, '夜樱', 2, '');
INSERT INTO `garden_maps` VALUES (25, 9, '雪莲花', 0, '');
INSERT INTO `garden_maps` VALUES (26, 9, '金雪莲', 1, '');
INSERT INTO `garden_maps` VALUES (27, 9, '七彩雪莲', 2, '');
INSERT INTO `garden_maps` VALUES (28, 10, '银色菊花', 0, '');
INSERT INTO `garden_maps` VALUES (29, 11, '银野花', 0, '');
INSERT INTO `garden_maps` VALUES (30, 12, '端阳花', 0, '');
INSERT INTO `garden_maps` VALUES (31, 13, '银友谊花', 0, '');
INSERT INTO `garden_maps` VALUES (32, 14, '银色烈焰焚情', 1, '');
INSERT INTO `garden_maps` VALUES (33, 15, '金色烈焰焚情', 2, '');

-- ----------------------------
-- Table structure for garden_mixes
-- ----------------------------
DROP TABLE IF EXISTS `garden_mixes`;
CREATE TABLE `garden_mixes`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `seed_id` bigint UNSIGNED NULL DEFAULT NULL,
  `flower` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `need` bigint NULL DEFAULT 1,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_mixes_seed_id`(`seed_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 11 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_mixes
-- ----------------------------
INSERT INTO `garden_mixes` VALUES (1, 10, '向日葵', 5);
INSERT INTO `garden_mixes` VALUES (2, 11, '向日葵', 3);
INSERT INTO `garden_mixes` VALUES (3, 11, '玫瑰花', 2);
INSERT INTO `garden_mixes` VALUES (4, 12, '玫瑰花', 4);
INSERT INTO `garden_mixes` VALUES (5, 12, '郁金香', 2);
INSERT INTO `garden_mixes` VALUES (6, 13, '向日葵', 2);
INSERT INTO `garden_mixes` VALUES (7, 13, '玫瑰花', 2);
INSERT INTO `garden_mixes` VALUES (8, 13, '郁金香', 2);
INSERT INTO `garden_mixes` VALUES (9, 14, '月光花', 3);
INSERT INTO `garden_mixes` VALUES (10, 15, '月光花', 6);

-- ----------------------------
-- Table structure for garden_msgs
-- ----------------------------
DROP TABLE IF EXISTS `garden_msgs`;
CREATE TABLE `garden_msgs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `uid` bigint UNSIGNED NULL DEFAULT NULL,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `fid` bigint UNSIGNED NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_msgs_uid`(`uid` ASC) USING BTREE,
  INDEX `idx_garden_msgs_f_id`(`fid` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 18 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_msgs
-- ----------------------------
INSERT INTO `garden_msgs` VALUES (1, 10000, '来花园帮忙浇水。', '2026-09-06 21:51:42.793', 10000);
INSERT INTO `garden_msgs` VALUES (2, 10000, '点亮了太阳神花图谱', '2026-09-06 21:51:56.634', 10000);
INSERT INTO `garden_msgs` VALUES (3, 10000, '来花园摘走了1朵红玫瑰。', '2026-09-06 21:54:43.328', 10001);
INSERT INTO `garden_msgs` VALUES (4, 10000, '送了您1朵红玫瑰', '2026-09-06 21:55:23.958', 10001);
INSERT INTO `garden_msgs` VALUES (5, 10007, '来花园帮忙浇水。', '2026-09-06 23:33:46.387', 10007);
INSERT INTO `garden_msgs` VALUES (6, 10007, '来花园帮忙浇水。', '2026-09-06 23:33:47.134', 10007);
INSERT INTO `garden_msgs` VALUES (7, 10007, '来花园帮忙锄草。', '2026-09-06 23:35:07.399', 10007);
INSERT INTO `garden_msgs` VALUES (8, 10007, '来花园帮忙锄草。', '2026-09-06 23:35:08.005', 10007);
INSERT INTO `garden_msgs` VALUES (9, 10007, '来花园帮忙捉虫。', '2026-09-06 23:35:35.866', 10007);
INSERT INTO `garden_msgs` VALUES (10, 10007, '来花园帮忙捉虫。', '2026-09-06 23:35:36.489', 10007);
INSERT INTO `garden_msgs` VALUES (11, 10007, '点亮了太阳神花图谱', '2026-09-06 23:37:29.822', 10007);
INSERT INTO `garden_msgs` VALUES (12, 10007, '种出了金色向日葵花朵', '2026-09-06 23:37:30.860', 10007);
INSERT INTO `garden_msgs` VALUES (13, 10007, '来花园帮忙浇水。', '2026-09-07 09:52:32.099', 10007);
INSERT INTO `garden_msgs` VALUES (14, 10007, '种出了蓝色妖姬花朵', '2026-09-07 10:21:04.459', 10007);
INSERT INTO `garden_msgs` VALUES (15, 10007, '点亮了粉郁金香图谱', '2026-09-07 10:21:06.169', 10007);
INSERT INTO `garden_msgs` VALUES (16, 10007, '点亮了黑郁金香图谱', '2026-09-07 12:08:36.557', 10007);
INSERT INTO `garden_msgs` VALUES (17, 10007, '点亮了七彩向日葵图谱', '2026-09-07 12:08:37.744', 10007);

-- ----------------------------
-- Table structure for garden_plots
-- ----------------------------
DROP TABLE IF EXISTS `garden_plots`;
CREATE TABLE `garden_plots`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `plot` bigint NULL DEFAULT NULL,
  `crop` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `seed_at` datetime(3) NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `seed_id` bigint UNSIGNED NULL DEFAULT 0,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `drys` bigint NULL DEFAULT 0,
  `weed` bigint NULL DEFAULT 0,
  `pest` bigint NULL DEFAULT 0,
  `yield` bigint NULL DEFAULT 0,
  `amount` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_garden_plots_user_id`(`user_id` ASC) USING BTREE,
  INDEX `idx_garden_plots_plot`(`plot` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 17 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_plots
-- ----------------------------
INSERT INTO `garden_plots` VALUES (1, 10001, 0, '', '2026-09-06 21:54:31.008', 2, '2026-09-04 11:42:43.663', '2026-09-06 21:54:43.315', 2, '红玫瑰', 1, 1, 1, 5, 4);
INSERT INTO `garden_plots` VALUES (2, 10001, 1, '向日葵', '2026-09-04 13:00:57.933', 0, '2026-09-04 11:42:43.667', '2026-09-06 21:54:30.976', 0, '', 0, 0, 0, 0, 0);
INSERT INTO `garden_plots` VALUES (5, 10007, 0, '', NULL, 0, '2026-09-04 11:51:05.513', '2026-09-07 12:08:36.566', 0, '', 0, 0, 0, 0, 0);
INSERT INTO `garden_plots` VALUES (6, 10007, 1, '', NULL, 0, '2026-09-04 11:51:05.519', '2026-09-07 12:08:37.754', 0, '', 0, 0, 0, 0, 0);
INSERT INTO `garden_plots` VALUES (11, 10000, 0, '向日葵', '2026-09-07 09:08:47.222', 2, '2026-09-05 23:13:20.299', '2026-09-07 09:17:31.334', 1, '金色向日葵', 0, 0, 0, 1, 1);
INSERT INTO `garden_plots` VALUES (12, 10000, 1, '', '2026-09-06 22:24:34.813', 2, '2026-09-05 23:13:20.306', '2026-09-06 23:40:11.637', 1, '金色向日葵', 0, 0, 0, 1, 1);
INSERT INTO `garden_plots` VALUES (15, 10007, 2, NULL, NULL, 0, '2026-09-07 10:21:12.564', '2026-09-07 12:08:39.450', 0, '', 0, 0, 0, 0, 0);
INSERT INTO `garden_plots` VALUES (16, 10007, 3, NULL, NULL, 0, '2026-09-07 15:38:05.340', '2026-09-07 15:38:05.340', 0, '', 0, 0, 0, 0, 0);

-- ----------------------------
-- Table structure for garden_seeds
-- ----------------------------
DROP TABLE IF EXISTS `garden_seeds`;
CREATE TABLE `garden_seeds`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `d_type` bigint NULL DEFAULT 0,
  `level` bigint NULL DEFAULT 1,
  `price` bigint NULL DEFAULT 0,
  `seed` bigint NULL DEFAULT 1,
  `ling` bigint NULL DEFAULT 1,
  `buds` bigint NULL DEFAULT 1,
  `less` bigint NULL DEFAULT 2,
  `more` bigint NULL DEFAULT 4,
  `status` bigint NULL DEFAULT 1,
  `remark` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 26 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_seeds
-- ----------------------------
INSERT INTO `garden_seeds` VALUES (1, '向日葵', 0, 1, 5, 1, 1, 1, 2, 4, 1, '沉默的爱，勇敢追求幸福。');
INSERT INTO `garden_seeds` VALUES (2, '玫瑰花', 0, 1, 10, 1, 1, 2, 3, 5, 1, '爱情与热恋，勇敢表达。');
INSERT INTO `garden_seeds` VALUES (3, '郁金香', 0, 2, 20, 2, 2, 2, 3, 6, 1, '博爱、体贴、高雅。');
INSERT INTO `garden_seeds` VALUES (4, '月光花', 0, 3, 40, 2, 2, 3, 4, 8, 1, '幸福与美好的憧憬。');
INSERT INTO `garden_seeds` VALUES (5, '百合花', 0, 4, 80, 3, 3, 3, 5, 10, 1, '百年好合，纯洁无瑕。');
INSERT INTO `garden_seeds` VALUES (6, '牡丹', 0, 5, 150, 3, 4, 4, 6, 12, 1, '圆满、浓情、富贵。');
INSERT INTO `garden_seeds` VALUES (7, '蓝色妖姬', 0, 7, 300, 4, 5, 5, 8, 16, 1, '奇迹与不可能的爱。');
INSERT INTO `garden_seeds` VALUES (8, '樱花', 0, 9, 500, 5, 6, 6, 10, 20, 1, '生命、幸福、热烈。');
INSERT INTO `garden_seeds` VALUES (9, '天山雪莲', 0, 12, 800, 6, 8, 8, 12, 24, 1, '纯洁的爱、坚贞。');
INSERT INTO `garden_seeds` VALUES (10, '银色菊花', 1, 3, 0, 2, 2, 2, 4, 8, 1, '真诚的思念。');
INSERT INTO `garden_seeds` VALUES (11, '银野花', 1, 4, 0, 2, 3, 3, 5, 10, 1, '野性之美。');
INSERT INTO `garden_seeds` VALUES (12, '端阳花', 1, 5, 0, 3, 3, 3, 6, 12, 1, '端午安康。');
INSERT INTO `garden_seeds` VALUES (13, '银友谊花', 1, 6, 0, 3, 4, 4, 7, 14, 1, '友谊长存。');
INSERT INTO `garden_seeds` VALUES (14, '银色烈焰焚情', 1, 8, 0, 4, 5, 5, 8, 16, 1, '炽热的爱。');
INSERT INTO `garden_seeds` VALUES (15, '金色烈焰焚情', 1, 10, 0, 5, 6, 6, 10, 20, 1, '永恒的爱。');
INSERT INTO `garden_seeds` VALUES (18, '染色药水', 2, 1, 5000, 1, 1, 1, 2, 4, 1, '可以给花朵染色。');
INSERT INTO `garden_seeds` VALUES (19, '魔力播种机', 2, 1, 20000, 1, 1, 1, 2, 4, 1, '一键播种所有空花盆。');
INSERT INTO `garden_seeds` VALUES (20, '魔力爱心棒', 2, 1, 20000, 1, 1, 1, 2, 4, 1, '一键照料所有花朵。');
INSERT INTO `garden_seeds` VALUES (21, '魔力收割机', 2, 1, 20000, 1, 1, 1, 2, 4, 1, '一键收获所有成熟花朵。');
INSERT INTO `garden_seeds` VALUES (22, '愿望果实', 2, 1, 5000, 1, 1, 1, 2, 4, 1, '许下一个美好的愿望。');
INSERT INTO `garden_seeds` VALUES (23, '小魔法花肥', 2, 1, 10000, 1, 1, 1, 2, 4, 1, '缩短花朵成长时间。');
INSERT INTO `garden_seeds` VALUES (24, '魔法营养液', 2, 1, 50000, 1, 1, 1, 2, 4, 1, '大幅缩短花朵成长时间。');
INSERT INTO `garden_seeds` VALUES (25, '小魔力营养液', 2, 1, 100000, 1, 1, 1, 2, 4, 1, '让花朵立即成熟。');

-- ----------------------------
-- Table structure for garden_signs
-- ----------------------------
DROP TABLE IF EXISTS `garden_signs`;
CREATE TABLE `garden_signs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sign_date` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `week_day` bigint NULL DEFAULT 0,
  `day_no` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `uk_gs`(`user_id` ASC, `sign_date` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of garden_signs
-- ----------------------------
INSERT INTO `garden_signs` VALUES (1, 10000, '2026-09-07', 1, 1, '2026-09-07 09:37:26.519');
INSERT INTO `garden_signs` VALUES (2, 10007, '2026-09-07', 1, 1, '2026-09-07 09:50:11.466');

-- ----------------------------
-- Table structure for gardens
-- ----------------------------
DROP TABLE IF EXISTS `gardens`;
CREATE TABLE `gardens`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `level` bigint NULL DEFAULT 1,
  `point` bigint NULL DEFAULT 0,
  `lands` bigint NULL DEFAULT 2,
  `notice` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `common` bigint NULL DEFAULT 0,
  `festival` bigint NULL DEFAULT 0,
  `scarce` bigint NULL DEFAULT 0,
  `basket_cnt` bigint NULL DEFAULT 0,
  `bottle_cnt` bigint NULL DEFAULT 0,
  `config` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_gardens_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of gardens
-- ----------------------------
INSERT INTO `gardens` VALUES (1, 10000, '站长小Q的秘密花园', 3, 129, 2, '勤劳致富，偷花可耻', 0, 0, 1, 0, 0, 1, '2026-09-06 21:48:19.386', '2026-09-07 09:37:26.528');
INSERT INTO `gardens` VALUES (2, 10007, '????', 3, 42, 4, '?????', 2, 2, 2, 37, 0, 1, '2026-09-06 21:54:03.939', '2026-09-07 15:38:05.389');
INSERT INTO `gardens` VALUES (3, 10001, '云起的花园', 1, 2, 2, '欢迎光临我的花园！', 0, 0, 0, 0, 1, 0, '2026-09-06 21:54:30.965', '2026-09-06 21:55:23.950');

-- ----------------------------
-- Table structure for goods
-- ----------------------------
DROP TABLE IF EXISTS `goods`;
CREATE TABLE `goods`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `desc` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `icon` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `category` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `price` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  `sort` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 11 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of goods
-- ----------------------------
INSERT INTO `goods` VALUES (1, '改名卡', '可修改一次昵称', '', '道具', 500, 1, 1);
INSERT INTO `goods` VALUES (2, '家园皮肤·蓝', '家园首页皮肤', '', '装扮', 1000, 1, 2);
INSERT INTO `goods` VALUES (3, '聊天气泡·金', '聊天室金色气泡', '', '装扮', 800, 1, 3);
INSERT INTO `goods` VALUES (4, '经验加速卡', '发帖回帖经验+50%', '', '特权', 1200, 1, 4);
INSERT INTO `goods` VALUES (5, '幸运星', '每日星运更亮', '', '道具', 300, 1, 5);
INSERT INTO `goods` VALUES (6, '头像框·玫瑰', '主页头像玫瑰框', '', '装扮', 600, 1, 6);
INSERT INTO `goods` VALUES (7, '玫瑰花', '娇艳欲滴的玫瑰，可在帖子下方送给好友', 'flower_rose.gif', '鲜花', 20, 1, 1);
INSERT INTO `goods` VALUES (8, '向日葵', '阳光灿烂的向日葵，送花示爱暖人心', 'flower_sun.gif', '鲜花', 15, 1, 2);
INSERT INTO `goods` VALUES (9, '郁金香', '高贵典雅的郁金香，送花祝福好运', 'flower_tulip.gif', '鲜花', 35, 1, 3);
INSERT INTO `goods` VALUES (10, '月光花', '月光下的神秘花朵，稀有珍品', 'flower_moon.gif', '鲜花', 60, 1, 4);

-- ----------------------------
-- Table structure for mood_comments
-- ----------------------------
DROP TABLE IF EXISTS `mood_comments`;
CREATE TABLE `mood_comments`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `mood_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_mood_comments_mood_id`(`mood_id` ASC) USING BTREE,
  INDEX `idx_mood_comments_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of mood_comments
-- ----------------------------

-- ----------------------------
-- Table structure for moods
-- ----------------------------
DROP TABLE IF EXISTS `moods`;
CREATE TABLE `moods`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_moods_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 13 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of moods
-- ----------------------------
INSERT INTO `moods` VALUES (1, 10007, '我的空间第一条', 0, '2026-08-30 22:40:39.414');
INSERT INTO `moods` VALUES (2, 10007, '转发：我的空间第一条', 0, '2026-08-30 22:40:40.735');
INSERT INTO `moods` VALUES (3, 10007, '转发：我的空间第一条', 0, '2026-08-30 22:40:48.078');
INSERT INTO `moods` VALUES (4, 10007, '转发：转发：我的空间第一条', 0, '2026-08-30 22:40:48.932');
INSERT INTO `moods` VALUES (5, 10007, '转发：转发：我的空间第一条', 0, '2026-08-30 22:40:49.542');
INSERT INTO `moods` VALUES (6, 10007, '转发：转发：转发：我的空间第一条', 0, '2026-08-30 22:40:49.698');
INSERT INTO `moods` VALUES (7, 10007, '转发：转发：转发：我的空间第一条', 0, '2026-08-30 22:40:49.890');
INSERT INTO `moods` VALUES (8, 10007, '转发：转发：转发：转发：我的空间第一条', 0, '2026-08-30 22:40:50.913');
INSERT INTO `moods` VALUES (9, 10007, '空间', 0, '2026-08-30 22:41:00.994');
INSERT INTO `moods` VALUES (10, 10000, '??????', 0, '2026-08-30 22:54:10.904');
INSERT INTO `moods` VALUES (11, 10007, '顶顶顶', 1, '2026-09-04 21:51:20.516');
INSERT INTO `moods` VALUES (12, 10001, '分享好帖：《【公坛区域】管理须知》>> /thread/10013', 1, '2026-09-05 15:54:17.507');

-- ----------------------------
-- Table structure for my_games
-- ----------------------------
DROP TABLE IF EXISTS `my_games`;
CREATE TABLE `my_games`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `game_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_mygame`(`user_id` ASC, `game_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 6 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of my_games
-- ----------------------------
INSERT INTO `my_games` VALUES (1, 10001, 1, '2026-09-04 12:20:06.169');
INSERT INTO `my_games` VALUES (3, 10001, 3, '2026-09-05 15:43:04.579');
INSERT INTO `my_games` VALUES (4, 10000, 3, '2026-09-05 23:13:19.109');
INSERT INTO `my_games` VALUES (5, 10007, 3, '2026-09-06 21:57:25.099');

-- ----------------------------
-- Table structure for noble_plans
-- ----------------------------
DROP TABLE IF EXISTS `noble_plans`;
CREATE TABLE `noble_plans`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `type` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `cost` bigint NULL DEFAULT 0,
  `gain` bigint NULL DEFAULT 0,
  `days` bigint NULL DEFAULT 0,
  `sort` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 5 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of noble_plans
-- ----------------------------
INSERT INTO `noble_plans` VALUES (1, 'blue', '包月蓝钻等级加速', 500, 100, 30, 1);
INSERT INTO `noble_plans` VALUES (2, 'qq', '包月超Q等级加速', 1000, 200, 30, 2);
INSERT INTO `noble_plans` VALUES (3, 'blue', '年费蓝钻等级飞速', 5000, 1200, 365, 3);
INSERT INTO `noble_plans` VALUES (4, 'qq', '年费超Q等级飞速', 8000, 2400, 365, 4);

-- ----------------------------
-- Table structure for notifications
-- ----------------------------
DROP TABLE IF EXISTS `notifications`;
CREATE TABLE `notifications`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `content` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `ref_id` bigint UNSIGNED NULL DEFAULT NULL,
  `is_read` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_notifications_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 20 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of notifications
-- ----------------------------
INSERT INTO `notifications` VALUES (2, 10002, 'friend', '云起 请求加你为好友', '到「好友」页面处理这条申请吧', 10001, 0, '2026-08-30 10:08:24.517');
INSERT INTO `notifications` VALUES (3, 10001, 'friend', '安珞 同意了你的好友申请', '你们已成为好友，去打个招呼吧', 10002, 0, '2026-08-30 10:08:25.010');
INSERT INTO `notifications` VALUES (4, 10004, 'reply', '站长小Q 回复了你的帖子', '《又一个多月没来了》来了新回复，快去看看吧', 10001, 0, '2026-08-30 10:28:38.181');
INSERT INTO `notifications` VALUES (5, 10006, 'system', '欢迎来到3GQQ家园社区', '你的家园号码是 10006，请牢记！新人礼包100金币已到账。多逛论坛多回帖，经验等级蹭蹭涨。', 0, 0, '2026-08-30 10:33:09.741');
INSERT INTO `notifications` VALUES (6, 10007, 'system', '欢迎来到3GQQ家园社区', '你的家园号码是 10007，请牢记！新人礼包100金币已到账。多逛论坛多回帖，经验等级蹭蹭涨。', 0, 1, '2026-08-30 10:36:20.879');
INSERT INTO `notifications` VALUES (7, 10002, 'reply', '夜凌云 回复了你的帖子', '《id10272+霓裳羽衣+申请皇家贵族》来了新回复，快去看看吧', 10006, 0, '2026-08-30 10:37:17.453');
INSERT INTO `notifications` VALUES (8, 10000, 'friend', '夜凌云 请求加你为好友', '到「好友」页面处理这条申请吧', 10007, 1, '2026-08-30 10:37:33.623');
INSERT INTO `notifications` VALUES (9, 10005, 'reply', '夜凌云 回复了你的帖子', '《孤独的人无所谓》来了新回复，快去看看吧', 10005, 0, '2026-08-30 10:37:57.062');
INSERT INTO `notifications` VALUES (10, 10004, 'friend', '夜凌云 请求加你为好友', '到「好友」页面处理这条申请吧', 10007, 0, '2026-08-30 10:38:00.950');
INSERT INTO `notifications` VALUES (11, 10000, 'reply', '夜凌云 回复了你的帖子', '《【新手大全实用手册】》来了新回复，快去看看吧', 10000, 1, '2026-08-30 10:39:17.845');
INSERT INTO `notifications` VALUES (12, 35797804, 'friend', '夜凌云 请求加你为好友', '到「好友」页面处理这条申请吧', 10007, 0, '2026-08-30 14:01:04.922');
INSERT INTO `notifications` VALUES (13, 35797804, 'friend', '站长小Q 请求加你为好友', '到「好友」页面处理这条申请吧', 10000, 0, '2026-08-30 14:17:03.968');
INSERT INTO `notifications` VALUES (14, 10007, 'friend', '站长小Q 同意了你的好友申请', '你们已成为好友，去打个招呼吧', 10000, 0, '2026-08-30 19:43:17.533');
INSERT INTO `notifications` VALUES (15, 10000, 'reply', '夜凌云 回复了你的帖子', '《表情测试帖》来了新回复，快去看看吧', 10035, 0, '2026-09-04 13:35:26.893');
INSERT INTO `notifications` VALUES (16, 10000, 'system', '举报处理结果', '你的举报（编号1）已处理：核实后未发现违规，予以忽略', 0, 0, '2026-09-05 15:45:50.951');
INSERT INTO `notifications` VALUES (17, 10002, 'system', '打赏', '你的帖子《【与世无争·周末聚会】来家族大厅唠唠嗑》收到来自 云起 的 100 金币打赏', 0, 0, '2026-09-05 15:46:30.894');
INSERT INTO `notifications` VALUES (18, 35797804, 'system', '送花', '你的帖子《【断念阁】新成员入阁欢迎帖》收到 站长小Q 送出的 2 朵「玫瑰花」', 0, 0, '2026-09-05 16:07:55.641');
INSERT INTO `notifications` VALUES (19, 35797804, 'system', '打赏', '你的帖子《【断念阁】新成员入阁欢迎帖》收到来自 站长小Q 的 100 G币打赏', 0, 0, '2026-09-05 20:05:12.674');

-- ----------------------------
-- Table structure for permissions
-- ----------------------------
DROP TABLE IF EXISTS `permissions`;
CREATE TABLE `permissions`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_permissions_code`(`code` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 9 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of permissions
-- ----------------------------
INSERT INTO `permissions` VALUES (1, '后台访问', 'admin:access', '进入管理后台');
INSERT INTO `permissions` VALUES (2, '用户管理', 'user:manage', '封禁/解封/重置密码/分配角色');
INSERT INTO `permissions` VALUES (3, '板块管理', 'board:manage', '板块增删改');
INSERT INTO `permissions` VALUES (4, '帖子管理', 'thread:manage', '置顶/精华/删帖删回复');
INSERT INTO `permissions` VALUES (5, '公告管理', 'announcement:manage', '公告广播增删改');
INSERT INTO `permissions` VALUES (6, '角色权限管理', 'role:manage', '角色与权限分配');
INSERT INTO `permissions` VALUES (7, '马甲管理', 'badge:manage', '勋章/马甲增删与授予');
INSERT INTO `permissions` VALUES (8, '游戏管理', 'game:manage', '游戏大厅增删改');

-- ----------------------------
-- Table structure for photos
-- ----------------------------
DROP TABLE IF EXISTS `photos`;
CREATE TABLE `photos`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `album_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `file` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `caption` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_photos_album_id`(`album_id` ASC) USING BTREE,
  INDEX `idx_photos_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of photos
-- ----------------------------

-- ----------------------------
-- Table structure for plaza_sections
-- ----------------------------
DROP TABLE IF EXISTS `plaza_sections`;
CREATE TABLE `plaza_sections`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `key` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `enabled` bigint NULL DEFAULT 1,
  `sort` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_plaza_sections_key`(`key` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 15 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of plaza_sections
-- ----------------------------
INSERT INTO `plaza_sections` VALUES (1, 'welcome', '欢迎·在线', 1, 0);
INSERT INTO `plaza_sections` VALUES (2, 'greeting', '问候', 1, 1);
INSERT INTO `plaza_sections` VALUES (3, 'tongcheng', '同城推荐', 1, 2);
INSERT INTO `plaza_sections` VALUES (4, 'tv', '家园TV', 1, 3);
INSERT INTO `plaza_sections` VALUES (5, 'tt', 'T台秀', 1, 4);
INSERT INTO `plaza_sections` VALUES (6, 'joy', '欢乐坊', 1, 5);
INSERT INTO `plaza_sections` VALUES (7, 'playground', '游乐场', 1, 6);
INSERT INTO `plaza_sections` VALUES (8, 'newthread', '最新发帖', 1, 7);
INSERT INTO `plaza_sections` VALUES (9, 'newreply', '最新回帖', 1, 8);
INSERT INTO `plaza_sections` VALUES (10, 'channels', '频道(公共/同城/家族)', 1, 9);
INSERT INTO `plaza_sections` VALUES (11, 'chat', '聊天大厅', 1, 10);
INSERT INTO `plaza_sections` VALUES (12, 'service', '社区服务', 1, 11);
INSERT INTO `plaza_sections` VALUES (13, 'dynamics', '用户动态', 1, 12);
INSERT INTO `plaza_sections` VALUES (14, 'search', '搜搜', 1, 13);

-- ----------------------------
-- Table structure for private_messages
-- ----------------------------
DROP TABLE IF EXISTS `private_messages`;
CREATE TABLE `private_messages`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `sender_id` bigint UNSIGNED NULL DEFAULT NULL,
  `receiver_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `is_read` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_private_messages_sender_id`(`sender_id` ASC) USING BTREE,
  INDEX `idx_private_messages_receiver_id`(`receiver_id` ASC) USING BTREE,
  CONSTRAINT `fk_private_messages_sender` FOREIGN KEY (`sender_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB AUTO_INCREMENT = 6 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of private_messages
-- ----------------------------
INSERT INTO `private_messages` VALUES (2, 10007, 10003, '你好', 0, '2026-08-30 10:36:40.725');
INSERT INTO `private_messages` VALUES (3, 10007, 10000, '你好', 1, '2026-08-30 10:37:38.026');
INSERT INTO `private_messages` VALUES (4, 10007, 35797804, 'aa', 0, '2026-08-30 14:01:24.099');
INSERT INTO `private_messages` VALUES (5, 10007, 35797804, '111', 0, '2026-09-04 21:06:08.038');

-- ----------------------------
-- Table structure for replies
-- ----------------------------
DROP TABLE IF EXISTS `replies`;
CREATE TABLE `replies`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `thread_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `floor` bigint NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `like_count` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_replies_thread_id`(`thread_id` ASC) USING BTREE,
  INDEX `idx_replies_user_id`(`user_id` ASC) USING BTREE,
  CONSTRAINT `fk_replies_thread` FOREIGN KEY (`thread_id`) REFERENCES `threads` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `fk_replies_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB AUTO_INCREMENT = 68 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of replies
-- ----------------------------
INSERT INTO `replies` VALUES (1, 10000, 10001, '感谢站长整理，收藏了！', 2, 1, '2026-08-29 01:07:01.677', 0);
INSERT INTO `replies` VALUES (2, 10000, 10002, '新人报到，学习学习~', 3, 1, '2026-08-29 06:07:01.677', 0);
INSERT INTO `replies` VALUES (3, 10000, 10005, '手册很实用，赞一个', 4, 1, '2026-08-29 07:07:01.677', 0);
INSERT INTO `replies` VALUES (4, 10001, 10001, '欢迎回来！老友', 2, 1, '2026-08-29 13:07:01.677', 0);
INSERT INTO `replies` VALUES (5, 10001, 10002, '抢车位！当年我买了十辆劳斯莱斯', 3, 1, '2026-08-28 13:07:01.677', 0);
INSERT INTO `replies` VALUES (6, 10001, 10003, '好友买卖才是永远的神', 4, 1, '2026-08-29 04:07:01.677', 0);
INSERT INTO `replies` VALUES (7, 10001, 10000, '魔法花园还有人记得吗，天天浇水', 5, 1, '2026-08-28 13:07:01.677', 0);
INSERT INTO `replies` VALUES (8, 10002, 10002, '【点歌】晴天+送给三年前的自己', 2, 1, '2026-08-28 15:07:01.677', 0);
INSERT INTO `replies` VALUES (9, 10002, 10005, '【点歌】七里香+祝家园越来越好', 3, 1, '2026-08-28 15:07:01.677', 0);
INSERT INTO `replies` VALUES (10, 10002, 10001, '收到，晚上开唱', 4, 1, '2026-08-29 13:07:01.677', 0);
INSERT INTO `replies` VALUES (11, 10003, 10001, '第三张绝了', 2, 1, '2026-08-29 19:07:01.677', 0);
INSERT INTO `replies` VALUES (12, 10003, 10004, '已取图，谢谢姐妹', 3, 1, '2026-08-29 10:07:01.677', 0);
INSERT INTO `replies` VALUES (13, 10004, 10003, '厦门报到', 2, 1, '2026-08-29 22:07:01.677', 0);
INSERT INTO `replies` VALUES (14, 10004, 10000, '福州报到，老乡好', 3, 1, '2026-08-29 10:07:01.677', 0);
INSERT INTO `replies` VALUES (15, 10005, 10004, '抱一个，都不容易', 2, 1, '2026-08-29 10:07:01.677', 0);
INSERT INTO `replies` VALUES (16, 10006, 10000, '材料齐全，予以通过，欢迎霓裳羽衣家族入驻！', 2, 1, '2026-08-27 19:07:01.677', 0);
INSERT INTO `replies` VALUES (17, 10007, 10000, '建议已收录，感谢你对家园的建议！', 2, 1, '2026-08-28 10:07:01.677', 0);
INSERT INTO `replies` VALUES (18, 10008, 10005, '华为老机型也适配，良心', 2, 1, '2026-08-29 01:07:01.677', 0);
INSERT INTO `replies` VALUES (20, 10001, 10000, '楼上说得对，好友买卖和抢车位真的是当年的快乐源泉，什么时候把游戏也复刻回来呀~', 6, 1, '2026-08-30 10:28:38.165', 0);
INSERT INTO `replies` VALUES (21, 10006, 10007, '哈哈', 3, 1, '2026-08-30 10:37:17.438', 0);
INSERT INTO `replies` VALUES (22, 10005, 10007, '对啊', 3, 1, '2026-08-30 10:37:57.046', 0);
INSERT INTO `replies` VALUES (23, 10000, 10007, '赞一个', 5, 1, '2026-08-30 10:39:17.831', 0);
INSERT INTO `replies` VALUES (24, 10009, 10001, '【报名】云起，常用长枪！', 2, 1, '2026-08-30 11:33:44.046', 0);
INSERT INTO `replies` VALUES (25, 10009, 10001, '已报名，求虐', 3, 1, '2026-08-30 10:33:44.052', 0);
INSERT INTO `replies` VALUES (26, 10010, 10001, '我的蓝玫瑰呢，先占楼', 2, 1, '2026-08-30 11:33:44.071', 0);
INSERT INTO `replies` VALUES (27, 10011, 10001, '楼主的幻影被我贴条了哈哈', 2, 1, '2026-08-30 11:33:44.090', 0);
INSERT INTO `replies` VALUES (28, 10012, 10001, '新区见！老玩家回归', 2, 1, '2026-08-30 11:33:44.112', 0);
INSERT INTO `replies` VALUES (29, 10013, 10001, '收到，各版版主学习一下！', 2, 1, '2026-08-30 13:44:10.076', 0);
INSERT INTO `replies` VALUES (43, 10016, 10001, '严厉支持！共同维护家园环境。', 2, 1, '2026-07-01 10:00:00.000', 0);
INSERT INTO `replies` VALUES (44, 10016, 35804196, '支持！见到一个举报一个。', 3, 1, '2026-07-02 09:00:00.000', 0);
INSERT INTO `replies` VALUES (45, 10017, 35806077, '收藏了，排版果然重要。', 2, 1, '2026-05-21 09:00:00.000', 0);
INSERT INTO `replies` VALUES (46, 10017, 35806205, '好用！帖子瞬间变好看了', 3, 1, '2026-05-21 20:00:00.000', 0);
INSERT INTO `replies` VALUES (47, 10017, 35803370, '小白福音，感谢未央大佬。', 4, 1, '2026-05-22 12:00:00.000', 0);
INSERT INTO `replies` VALUES (48, 10018, 10000, '请到客服中心发帖申诉，管理员核实后帮你重置密码。', 2, 1, '2026-08-21 09:00:00.000', 0);
INSERT INTO `replies` VALUES (49, 10019, 35799015, '版规收到，每天来打卡！', 2, 1, '2026-06-16 08:30:00.000', 0);
INSERT INTO `replies` VALUES (50, 10020, 10003, '好诗！有田园气息。', 2, 1, '2026-08-29 11:30:00.000', 0);
INSERT INTO `replies` VALUES (51, 10020, 10002, '来晚了，赏了~', 3, 1, '2026-08-29 12:00:00.000', 0);
INSERT INTO `replies` VALUES (52, 10025, 35805768, '梦醒时分，各自安好。', 2, 1, '2026-08-26 09:00:00.000', 0);
INSERT INTO `replies` VALUES (53, 10025, 35805532, '字字扎心，抱一个。', 3, 1, '2026-08-26 20:00:00.000', 0);
INSERT INTO `replies` VALUES (54, 10026, 10002, '祝百年好合！家园第一对？', 2, 1, '2026-08-23 10:00:00.000', 0);
INSERT INTO `replies` VALUES (55, 10028, 35792031, '家族有你真好。', 2, 1, '2026-08-06 09:00:00.000', 0);
INSERT INTO `replies` VALUES (56, 10028, 35806199, '那19天是家族最热闹的时候。', 3, 1, '2026-08-06 12:00:00.000', 0);
INSERT INTO `replies` VALUES (57, 10029, 35803370, '恭喜恭喜！向大咪学习。', 2, 1, '2026-08-19 08:00:00.000', 0);
INSERT INTO `replies` VALUES (58, 10029, 35799015, '撒花✿', 3, 1, '2026-08-19 10:00:00.000', 0);
INSERT INTO `replies` VALUES (59, 10030, 35797804, '材料收到，公坛已受理，等待审核。', 2, 1, '2026-08-09 09:00:00.000', 0);
INSERT INTO `replies` VALUES (60, 10031, 35804196, '我选大电池，一天三充受不了。', 2, 1, '2026-08-26 14:00:00.000', 0);
INSERT INTO `replies` VALUES (61, 10032, 35803370, '等一个真香价。', 2, 1, '2026-08-24 11:00:00.000', 0);
INSERT INTO `replies` VALUES (62, 10033, 10001, '纲要已读，共同遵守！', 2, 1, '2026-07-15 17:00:00.000', 0);
INSERT INTO `replies` VALUES (63, 10033, 35805083, '已转发客服团学习。', 3, 1, '2026-07-16 09:00:00.000', 0);
INSERT INTO `replies` VALUES (64, 10034, 10001, '大事务！恭喜家园。', 2, 1, '2026-06-10 12:00:00.000', 0);
INSERT INTO `replies` VALUES (65, 10034, 35805768, '普天同庆，家园长长久久。', 3, 1, '2026-06-10 14:00:00.000', 0);
INSERT INTO `replies` VALUES (66, 10009, 10000, '回复3楼：你很棒棒', 4, 1, '2026-08-30 14:44:40.034', 0);
INSERT INTO `replies` VALUES (67, 10035, 10007, '回复1楼：1111', 2, 1, '2026-09-04 13:35:26.880', 0);

-- ----------------------------
-- Table structure for reply_votes
-- ----------------------------
DROP TABLE IF EXISTS `reply_votes`;
CREATE TABLE `reply_votes`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `reply_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_rv`(`reply_id` ASC, `user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of reply_votes
-- ----------------------------

-- ----------------------------
-- Table structure for reports
-- ----------------------------
DROP TABLE IF EXISTS `reports`;
CREATE TABLE `reports`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `reporter_id` bigint UNSIGNED NULL DEFAULT NULL,
  `target_type` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `target_id` bigint UNSIGNED NULL DEFAULT NULL,
  `reason` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 0,
  `result` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `handler_id` bigint UNSIGNED NULL DEFAULT NULL,
  `handled_at` datetime(3) NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_reports_reporter_id`(`reporter_id` ASC) USING BTREE,
  INDEX `idx_reports_target_type`(`target_type` ASC) USING BTREE,
  INDEX `idx_reports_target_id`(`target_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of reports
-- ----------------------------
INSERT INTO `reports` VALUES (1, 10000, 'thread', 10038, '????', 1, '核实后未发现违规，予以忽略', 10000, '2026-09-05 15:45:50.946', '2026-09-05 15:45:39.396');

-- ----------------------------
-- Table structure for resources
-- ----------------------------
DROP TABLE IF EXISTS `resources`;
CREATE TABLE `resources`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `file` varchar(120) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `category` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `level` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_resources_file`(`file` ASC) USING BTREE,
  INDEX `idx_resources_category`(`category` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 520 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of resources
-- ----------------------------
INSERT INTO `resources` VALUES (1, 'picture/706.jpg', 'badge', '706.jpg', 0, 1);
INSERT INTO `resources` VALUES (2, 'picture/3.gif', 'badge', '3.gif', 0, 1);
INSERT INTO `resources` VALUES (3, 'picture/501.gif', 'badge', '501.gif', 0, 1);
INSERT INTO `resources` VALUES (4, 'picture/704.gif', 'badge', '704.gif', 0, 1);
INSERT INTO `resources` VALUES (5, 'picture/803.gif', 'badge', '803.gif', 0, 1);
INSERT INTO `resources` VALUES (6, 'picture/804.gif', 'badge', '804.gif', 0, 1);
INSERT INTO `resources` VALUES (7, 'picture/15.gif', 'badge', '15.gif', 0, 1);
INSERT INTO `resources` VALUES (8, 'picture/103.gif', 'badge', '103.gif', 0, 1);
INSERT INTO `resources` VALUES (9, 'picture/903.gif', 'badge', '903.gif', 0, 1);
INSERT INTO `resources` VALUES (10, 'picture/45.gif', 'badge', '45.gif', 0, 1);
INSERT INTO `resources` VALUES (11, 'picture/131851611.jpg', 'avatar', '131851611.jpg', 0, 1);
INSERT INTO `resources` VALUES (12, 'picture/1985acg.jpg', 'avatar', '1985acg.jpg', 0, 1);
INSERT INTO `resources` VALUES (13, 'picture/104039478.jpg', 'avatar', '104039478.jpg', 0, 1);
INSERT INTO `resources` VALUES (14, 'picture/125703412.png', 'avatar', '125703412.png', 0, 1);
INSERT INTO `resources` VALUES (15, 'picture/125751186.gif', 'avatar', '125751186.gif', 0, 1);
INSERT INTO `resources` VALUES (16, 'picture/1031047330.png', 'avatar', '1031047330.png', 0, 1);
INSERT INTO `resources` VALUES (17, 'image/logo.jpg', 'game', 'logo.jpg', 0, 1);
INSERT INTO `resources` VALUES (18, 'image/mofahuayuan.gif', 'game', 'mofahuayuan.gif', 0, 1);
INSERT INTO `resources` VALUES (19, 'image/hunli2.jpg', 'game', 'hunli2.jpg', 0, 1);
INSERT INTO `resources` VALUES (20, 'image/kaixinnongchang.gif', 'game', 'kaixinnongchang.gif', 0, 1);
INSERT INTO `resources` VALUES (21, 'image/kuangqiangchewei.gif', 'game', 'kuangqiangchewei.gif', 0, 1);
INSERT INTO `resources` VALUES (22, 'image/jwt.png', 'game', 'jwt.png', 0, 1);
INSERT INTO `resources` VALUES (23, 'image/cwlogo.gif', 'game', 'cwlogo.gif', 0, 1);
INSERT INTO `resources` VALUES (24, 'image/shuiguoleyuan.gif', 'game', 'shuiguoleyuan.gif', 0, 1);
INSERT INTO `resources` VALUES (25, 'image/quanminliema.gif', 'game', 'quanminliema.gif', 0, 1);
INSERT INTO `resources` VALUES (26, 'image/jiayuangushi.gif', 'game', 'jiayuangushi.gif', 0, 1);
INSERT INTO `resources` VALUES (27, 'image/dahuachuiniu.gif', 'game', 'dahuachuiniu.gif', 0, 1);
INSERT INTO `resources` VALUES (36, 'picture/0.gif', 'other', '0', 0, 1);
INSERT INTO `resources` VALUES (37, 'picture/001.gif', 'other', '001', 0, 1);
INSERT INTO `resources` VALUES (38, 'picture/001130576.jpeg', 'other', '001130576', 0, 1);
INSERT INTO `resources` VALUES (39, 'picture/001930850.jpg', 'other', '001930850', 0, 1);
INSERT INTO `resources` VALUES (40, 'picture/005949237.jpg', 'other', '005949237', 0, 1);
INSERT INTO `resources` VALUES (41, 'picture/012140779.jpg', 'other', '012140779', 0, 1);
INSERT INTO `resources` VALUES (42, 'picture/0224115780.gif', 'other', '0224115780', 0, 1);
INSERT INTO `resources` VALUES (43, 'picture/0224115784.gif', 'other', '0224115784', 0, 1);
INSERT INTO `resources` VALUES (44, 'picture/0227264740.jpeg', 'other', '0227264740', 0, 1);
INSERT INTO `resources` VALUES (45, 'picture/0230391366.gif', 'other', '0230391366', 0, 1);
INSERT INTO `resources` VALUES (46, 'picture/030300807.gif', 'other', '030300807', 0, 1);
INSERT INTO `resources` VALUES (47, 'picture/0348524645.gif', 'other', '0348524645', 0, 1);
INSERT INTO `resources` VALUES (48, 'picture/0348524646.gif', 'other', '0348524646', 0, 1);
INSERT INTO `resources` VALUES (49, 'picture/065449760.jpg', 'other', '065449760', 0, 1);
INSERT INTO `resources` VALUES (50, 'picture/0704380770.jpg', 'other', '0704380770', 0, 1);
INSERT INTO `resources` VALUES (51, 'picture/080650133.jpg', 'other', '080650133', 0, 1);
INSERT INTO `resources` VALUES (52, 'picture/094248484.jpg', 'other', '094248484', 0, 1);
INSERT INTO `resources` VALUES (53, 'picture/1.2.gif', 'other', '1.2', 0, 1);
INSERT INTO `resources` VALUES (54, 'picture/1.2.png', 'other', '1.2', 0, 1);
INSERT INTO `resources` VALUES (55, 'picture/1.gif', 'other', '1', 0, 1);
INSERT INTO `resources` VALUES (56, 'picture/10.gif', 'other', '10', 0, 1);
INSERT INTO `resources` VALUES (57, 'picture/100.gif', 'other', '100', 0, 1);
INSERT INTO `resources` VALUES (58, 'picture/101.gif', 'other', '101', 0, 1);
INSERT INTO `resources` VALUES (59, 'picture/102.gif', 'other', '102', 0, 1);
INSERT INTO `resources` VALUES (60, 'picture/102557044.jpg', 'other', '102557044', 0, 1);
INSERT INTO `resources` VALUES (61, 'picture/1031.gif', 'other', '1031', 0, 1);
INSERT INTO `resources` VALUES (62, 'picture/1031047331.png', 'other', '1031047331', 0, 1);
INSERT INTO `resources` VALUES (63, 'picture/104713396.jpg', 'other', '104713396', 0, 1);
INSERT INTO `resources` VALUES (64, 'picture/106.gif', 'other', '106', 0, 1);
INSERT INTO `resources` VALUES (65, 'picture/107.gif', 'other', '107', 0, 1);
INSERT INTO `resources` VALUES (66, 'picture/11.gif', 'other', '11', 0, 1);
INSERT INTO `resources` VALUES (67, 'picture/110.gif', 'other', '110', 0, 1);
INSERT INTO `resources` VALUES (68, 'picture/110603645.gif', 'other', '110603645', 0, 1);
INSERT INTO `resources` VALUES (69, 'picture/110905174.jpeg', 'other', '110905174', 0, 1);
INSERT INTO `resources` VALUES (70, 'picture/111751396.gif', 'other', '111751396', 0, 1);
INSERT INTO `resources` VALUES (71, 'picture/113.gif', 'other', '113', 0, 1);
INSERT INTO `resources` VALUES (72, 'picture/113541092.jpeg', 'other', '113541092', 0, 1);
INSERT INTO `resources` VALUES (73, 'picture/113658318.jpg', 'other', '113658318', 0, 1);
INSERT INTO `resources` VALUES (74, 'picture/115614719.jpg', 'other', '115614719', 0, 1);
INSERT INTO `resources` VALUES (75, 'picture/115745347.jpg', 'other', '115745347', 0, 1);
INSERT INTO `resources` VALUES (76, 'picture/117.gif', 'other', '117', 0, 1);
INSERT INTO `resources` VALUES (77, 'picture/12.gif', 'other', '12', 0, 1);
INSERT INTO `resources` VALUES (78, 'picture/1203.gif', 'other', '1203', 0, 1);
INSERT INTO `resources` VALUES (79, 'picture/124.35793524.gif', 'other', '124.35793524', 0, 1);
INSERT INTO `resources` VALUES (80, 'picture/124.gif', 'other', '124', 0, 1);
INSERT INTO `resources` VALUES (81, 'picture/126.gif', 'other', '126', 0, 1);
INSERT INTO `resources` VALUES (82, 'picture/13.gif', 'other', '13', 0, 1);
INSERT INTO `resources` VALUES (83, 'picture/132328804.png', 'other', '132328804', 0, 1);
INSERT INTO `resources` VALUES (84, 'picture/14.gif', 'other', '14', 0, 1);
INSERT INTO `resources` VALUES (85, 'picture/145726054.jpg', 'other', '145726054', 0, 1);
INSERT INTO `resources` VALUES (86, 'picture/145803067.jpg', 'other', '145803067', 0, 1);
INSERT INTO `resources` VALUES (87, 'picture/152225473.jpeg', 'other', '152225473', 0, 1);
INSERT INTO `resources` VALUES (88, 'picture/16.gif', 'other', '16', 0, 1);
INSERT INTO `resources` VALUES (89, 'picture/161416104.jpg', 'other', '161416104', 0, 1);
INSERT INTO `resources` VALUES (90, 'picture/161425291.jpg', 'other', '161425291', 0, 1);
INSERT INTO `resources` VALUES (91, 'picture/161433354.jpg', 'other', '161433354', 0, 1);
INSERT INTO `resources` VALUES (92, 'picture/165529331.gif', 'other', '165529331', 0, 1);
INSERT INTO `resources` VALUES (93, 'picture/17.gif', 'other', '17', 0, 1);
INSERT INTO `resources` VALUES (94, 'picture/171315051.gif', 'other', '171315051', 0, 1);
INSERT INTO `resources` VALUES (95, 'picture/1726203700.gif', 'other', '1726203700', 0, 1);
INSERT INTO `resources` VALUES (96, 'picture/174135940.jpg', 'other', '174135940', 0, 1);
INSERT INTO `resources` VALUES (97, 'picture/174351649.png', 'other', '174351649', 0, 1);
INSERT INTO `resources` VALUES (98, 'picture/18.gif', 'other', '18', 0, 1);
INSERT INTO `resources` VALUES (99, 'picture/1824042600.gif', 'other', '1824042600', 0, 1);
INSERT INTO `resources` VALUES (100, 'picture/1829271700.jpg', 'other', '1829271700', 0, 1);
INSERT INTO `resources` VALUES (101, 'picture/1829271701.jpg', 'other', '1829271701', 0, 1);
INSERT INTO `resources` VALUES (102, 'picture/19.gif', 'other', '19', 0, 1);
INSERT INTO `resources` VALUES (103, 'picture/191223870.jpeg', 'other', '191223870', 0, 1);
INSERT INTO `resources` VALUES (104, 'picture/193516922.jpg', 'other', '193516922', 0, 1);
INSERT INTO `resources` VALUES (105, 'picture/1_0.gif', 'other', '1_0', 0, 1);
INSERT INTO `resources` VALUES (106, 'picture/1_10.gif', 'other', '1_10', 0, 1);
INSERT INTO `resources` VALUES (107, 'picture/1_13.gif', 'other', '1_13', 0, 1);
INSERT INTO `resources` VALUES (108, 'picture/1_19.gif', 'other', '1_19', 0, 1);
INSERT INTO `resources` VALUES (109, 'picture/1_37.gif', 'other', '1_37', 0, 1);
INSERT INTO `resources` VALUES (110, 'picture/2.gif', 'other', '2', 0, 1);
INSERT INTO `resources` VALUES (111, 'picture/2.png', 'other', '2', 0, 1);
INSERT INTO `resources` VALUES (112, 'picture/20.gif', 'other', '20', 0, 1);
INSERT INTO `resources` VALUES (113, 'picture/200.gif', 'other', '200', 0, 1);
INSERT INTO `resources` VALUES (114, 'picture/21.gif', 'other', '21', 0, 1);
INSERT INTO `resources` VALUES (115, 'picture/2104192400.png', 'other', '2104192400', 0, 1);
INSERT INTO `resources` VALUES (116, 'picture/211537768.gif', 'other', '211537768', 0, 1);
INSERT INTO `resources` VALUES (117, 'picture/213545670.jpg', 'other', '213545670', 0, 1);
INSERT INTO `resources` VALUES (118, 'picture/22.gif', 'other', '22', 0, 1);
INSERT INTO `resources` VALUES (119, 'picture/220228144.jpg', 'other', '220228144', 0, 1);
INSERT INTO `resources` VALUES (120, 'picture/221605930.jpg', 'other', '221605930', 0, 1);
INSERT INTO `resources` VALUES (121, 'picture/224128670.jpg', 'other', '224128670', 0, 1);
INSERT INTO `resources` VALUES (122, 'picture/23.gif', 'other', '23', 0, 1);
INSERT INTO `resources` VALUES (123, 'picture/2321102490.jpeg', 'other', '2321102490', 0, 1);
INSERT INTO `resources` VALUES (124, 'picture/232706275.jpeg', 'other', '232706275', 0, 1);
INSERT INTO `resources` VALUES (125, 'picture/233333701.png', 'other', '233333701', 0, 1);
INSERT INTO `resources` VALUES (126, 'picture/24.gif', 'other', '24', 0, 1);
INSERT INTO `resources` VALUES (127, 'picture/25.gif', 'other', '25', 0, 1);
INSERT INTO `resources` VALUES (128, 'picture/26.gif', 'other', '26', 0, 1);
INSERT INTO `resources` VALUES (129, 'picture/27.gif', 'other', '27', 0, 1);
INSERT INTO `resources` VALUES (130, 'picture/28.gif', 'other', '28', 0, 1);
INSERT INTO `resources` VALUES (131, 'picture/29.gif', 'other', '29', 0, 1);
INSERT INTO `resources` VALUES (132, 'picture/31.gif', 'other', '31', 0, 1);
INSERT INTO `resources` VALUES (133, 'picture/32.gif', 'other', '32', 0, 1);
INSERT INTO `resources` VALUES (134, 'picture/33.gif', 'other', '33', 0, 1);
INSERT INTO `resources` VALUES (135, 'picture/34.gif', 'other', '34', 0, 1);
INSERT INTO `resources` VALUES (136, 'picture/40.gif', 'other', '40', 0, 1);
INSERT INTO `resources` VALUES (137, 'picture/41.gif', 'other', '41', 0, 1);
INSERT INTO `resources` VALUES (138, 'picture/42.gif', 'other', '42', 0, 1);
INSERT INTO `resources` VALUES (139, 'picture/43.gif', 'other', '43', 0, 1);
INSERT INTO `resources` VALUES (140, 'picture/44.gif', 'other', '44', 0, 1);
INSERT INTO `resources` VALUES (141, 'picture/46.gif', 'other', '46', 0, 1);
INSERT INTO `resources` VALUES (142, 'picture/500.gif', 'other', '500', 0, 1);
INSERT INTO `resources` VALUES (143, 'picture/51.gif', 'other', '51', 0, 1);
INSERT INTO `resources` VALUES (144, 'picture/56.gif', 'other', '56', 0, 1);
INSERT INTO `resources` VALUES (145, 'picture/6.gif', 'other', '6', 0, 1);
INSERT INTO `resources` VALUES (146, 'picture/63.gif', 'other', '63', 0, 1);
INSERT INTO `resources` VALUES (147, 'picture/65.gif', 'other', '65', 0, 1);
INSERT INTO `resources` VALUES (148, 'picture/66.gif', 'other', '66', 0, 1);
INSERT INTO `resources` VALUES (149, 'picture/666.gif', 'other', '666', 0, 1);
INSERT INTO `resources` VALUES (150, 'picture/7.gif', 'other', '7', 0, 1);
INSERT INTO `resources` VALUES (151, 'picture/701.gif', 'other', '701', 0, 1);
INSERT INTO `resources` VALUES (152, 'picture/703.gif', 'other', '703', 0, 1);
INSERT INTO `resources` VALUES (153, 'picture/705.gif', 'other', '705', 0, 1);
INSERT INTO `resources` VALUES (154, 'picture/77.gif', 'other', '77', 0, 1);
INSERT INTO `resources` VALUES (155, 'picture/79.gif', 'other', '79', 0, 1);
INSERT INTO `resources` VALUES (156, 'picture/8.gif', 'other', '8', 0, 1);
INSERT INTO `resources` VALUES (157, 'picture/810.png', 'other', '810', 0, 1);
INSERT INTO `resources` VALUES (158, 'picture/815.gif', 'other', '815', 0, 1);
INSERT INTO `resources` VALUES (159, 'picture/83.gif', 'other', '83', 0, 1);
INSERT INTO `resources` VALUES (160, 'picture/9.gif', 'other', '9', 0, 1);
INSERT INTO `resources` VALUES (161, 'picture/90.gif', 'other', '90', 0, 1);
INSERT INTO `resources` VALUES (162, 'picture/904.gif', 'other', '904', 0, 1);
INSERT INTO `resources` VALUES (163, 'picture/91.gif', 'other', '91', 0, 1);
INSERT INTO `resources` VALUES (164, 'picture/94.gif', 'other', '94', 0, 1);
INSERT INTO `resources` VALUES (165, 'picture/96.gif', 'other', '96', 0, 1);
INSERT INTO `resources` VALUES (166, 'picture/97.gif', 'other', '97', 0, 1);
INSERT INTO `resources` VALUES (167, 'picture/9703.gif', 'other', '9703', 0, 1);
INSERT INTO `resources` VALUES (168, 'picture/98.gif', 'other', '98', 0, 1);
INSERT INTO `resources` VALUES (169, 'picture/Connect_logo_3.png', 'other', 'Connect_logo_3', 0, 1);
INSERT INTO `resources` VALUES (170, 'picture/active.gif', 'other', 'active', 0, 1);
INSERT INTO `resources` VALUES (171, 'picture/apex.gif', 'other', 'apex', 0, 1);
INSERT INTO `resources` VALUES (172, 'picture/api-dongman_302.jpg', 'other', 'api-dongman_302', 0, 1);
INSERT INTO `resources` VALUES (173, 'picture/api-dongman_images.jpg', 'other', 'api-dongman_images', 0, 1);
INSERT INTO `resources` VALUES (174, 'picture/api.jpg', 'other', 'api', 0, 1);
INSERT INTO `resources` VALUES (175, 'picture/api1.jpg', 'other', 'api1', 0, 1);
INSERT INTO `resources` VALUES (176, 'picture/b0.png', 'other', 'b0', 0, 1);
INSERT INTO `resources` VALUES (177, 'picture/b1.png', 'other', 'b1', 0, 1);
INSERT INTO `resources` VALUES (178, 'picture/b2.png', 'other', 'b2', 0, 1);
INSERT INTO `resources` VALUES (179, 'picture/cos-img.jpg', 'other', 'cos-img', 0, 1);
INSERT INTO `resources` VALUES (180, 'picture/docu.jpg', 'other', 'docu', 0, 1);
INSERT INTO `resources` VALUES (181, 'picture/hl_2.gif', 'other', 'hl_2', 0, 1);
INSERT INTO `resources` VALUES (182, 'picture/hunli2.jpg', 'other', 'hunli2', 0, 1);
INSERT INTO `resources` VALUES (183, 'picture/img.jj20.jpg', 'other', 'img.jj20', 0, 1);
INSERT INTO `resources` VALUES (184, 'picture/liang.gif', 'other', 'liang', 0, 1);
INSERT INTO `resources` VALUES (185, 'picture/mcapi.jpg', 'other', 'mcapi', 0, 1);
INSERT INTO `resources` VALUES (186, 'picture/no.gif', 'other', 'no', 0, 1);
INSERT INTO `resources` VALUES (187, 'picture/notice.bmp', 'other', 'notice', 0, 1);
INSERT INTO `resources` VALUES (188, 'picture/notice.gif', 'other', 'notice', 0, 1);
INSERT INTO `resources` VALUES (189, 'picture/phone.jpg', 'other', 'phone', 0, 1);
INSERT INTO `resources` VALUES (190, 'picture/picture.jpg', 'other', 'picture', 0, 1);
INSERT INTO `resources` VALUES (191, 'picture/picture1.jpg', 'other', 'picture1', 0, 1);
INSERT INTO `resources` VALUES (192, 'picture/picture2.jpg', 'other', 'picture2', 0, 1);
INSERT INTO `resources` VALUES (193, 'picture/random.jpg', 'other', 'random', 0, 1);
INSERT INTO `resources` VALUES (194, 'picture/random1.jpg', 'other', 'random1', 0, 1);
INSERT INTO `resources` VALUES (195, 'picture/random2.jpg', 'other', 'random2', 0, 1);
INSERT INTO `resources` VALUES (196, 'picture/random3.jpg', 'other', 'random3', 0, 1);
INSERT INTO `resources` VALUES (197, 'picture/recom.gif', 'other', 'recom', 0, 1);
INSERT INTO `resources` VALUES (198, 'picture/tree.gif', 'other', 'tree', 0, 1);
INSERT INTO `resources` VALUES (201, 'picture/v11.gif', 'other', 'v11', 0, 1);
INSERT INTO `resources` VALUES (203, 'picture/v14.gif', 'other', 'v14', 0, 1);
INSERT INTO `resources` VALUES (204, 'picture/v15.gif', 'other', 'v15', 0, 1);
INSERT INTO `resources` VALUES (206, 'picture/v17.gif', 'other', 'v17', 0, 1);
INSERT INTO `resources` VALUES (208, 'picture/v19.gif', 'other', 'v19', 0, 1);
INSERT INTO `resources` VALUES (209, 'picture/v20.gif', 'other', 'v20', 0, 1);
INSERT INTO `resources` VALUES (210, 'picture/v21.gif', 'other', 'v21', 0, 1);
INSERT INTO `resources` VALUES (211, 'picture/v22.gif', 'other', 'v22', 0, 1);
INSERT INTO `resources` VALUES (212, 'picture/v23.gif', 'other', 'v23', 0, 1);
INSERT INTO `resources` VALUES (216, 'picture/xiaojiejie1.jpg', 'other', 'xiaojiejie1', 0, 1);
INSERT INTO `resources` VALUES (217, 'picture/xiaojiejie2.jpg', 'other', 'xiaojiejie2', 0, 1);
INSERT INTO `resources` VALUES (218, 'picture/zmxy_2022.jpg', 'other', 'zmxy_2022', 0, 1);
INSERT INTO `resources` VALUES (219, 'picture/▒╕░╕═╝▒ъ.png', 'other', '▒╕░╕═╝▒ъ', 0, 1);
INSERT INTO `resources` VALUES (220, 'image/bar.gif', 'other', 'bar', 0, 1);
INSERT INTO `resources` VALUES (221, 'image/bg_for_ie.png', 'other', 'bg_for_ie', 0, 1);
INSERT INTO `resources` VALUES (222, 'image/bg_main_nav2.png', 'other', 'bg_main_nav2', 0, 1);
INSERT INTO `resources` VALUES (223, 'image/bg_module_content_list_dot_01.gif', 'other', 'bg_module_content_list_dot_01', 0, 1);
INSERT INTO `resources` VALUES (224, 'image/bg_selector.gif', 'other', 'bg_selector', 0, 1);
INSERT INTO `resources` VALUES (225, 'image/bg_selector.png', 'other', 'bg_selector', 0, 1);
INSERT INTO `resources` VALUES (226, 'image/bg_sharebox.gif', 'other', 'bg_sharebox', 0, 1);
INSERT INTO `resources` VALUES (227, 'image/bg_sharebox.png', 'other', 'bg_sharebox', 0, 1);
INSERT INTO `resources` VALUES (228, 'image/bg_tab_nav.png', 'other', 'bg_tab_nav', 0, 1);
INSERT INTO `resources` VALUES (229, 'image/bg_tips.png', 'other', 'bg_tips', 0, 1);
INSERT INTO `resources` VALUES (230, 'image/bg_tips_btm.png', 'other', 'bg_tips_btm', 0, 1);
INSERT INTO `resources` VALUES (231, 'image/bg_tips_top.png', 'other', 'bg_tips_top', 0, 1);
INSERT INTO `resources` VALUES (232, 'image/bg_trans.png', 'other', 'bg_trans', 0, 1);
INSERT INTO `resources` VALUES (233, 'image/bg_wb_uibody.png', 'other', 'bg_wb_uibody', 0, 1);
INSERT INTO `resources` VALUES (234, 'image/bg_wb_uiplus.gif', 'other', 'bg_wb_uiplus', 0, 1);
INSERT INTO `resources` VALUES (235, 'image/bg_wb_uiplus.png', 'other', 'bg_wb_uiplus', 0, 1);
INSERT INTO `resources` VALUES (236, 'image/btn_l_gray.gif', 'other', 'btn_l_gray', 0, 1);
INSERT INTO `resources` VALUES (237, 'image/btn_m_gray.gif', 'other', 'btn_m_gray', 0, 1);
INSERT INTO `resources` VALUES (238, 'image/btn_s_gray.gif', 'other', 'btn_s_gray', 0, 1);
INSERT INTO `resources` VALUES (239, 'image/btn_xl_gray.gif', 'other', 'btn_xl_gray', 0, 1);
INSERT INTO `resources` VALUES (240, 'image/btns_bg.png', 'other', 'btns_bg', 0, 1);
INSERT INTO `resources` VALUES (241, 'image/btns_word_share.png', 'other', 'btns_word_share', 0, 1);
INSERT INTO `resources` VALUES (242, 'image/checkbox-checked-disabled.png', 'other', 'checkbox-checked-disabled', 0, 1);
INSERT INTO `resources` VALUES (243, 'image/checkbox-checked.png', 'other', 'checkbox-checked', 0, 1);
INSERT INTO `resources` VALUES (244, 'image/checkbox-unchecked.png', 'other', 'checkbox-unchecked', 0, 1);
INSERT INTO `resources` VALUES (245, 'image/close_icon.gif', 'other', 'close_icon', 0, 1);
INSERT INTO `resources` VALUES (246, 'image/close_icon.png', 'other', 'close_icon', 0, 1);
INSERT INTO `resources` VALUES (247, 'image/close_icon1.png', 'other', 'close_icon1', 0, 1);
INSERT INTO `resources` VALUES (248, 'image/dlg_bg.png', 'other', 'dlg_bg', 0, 1);
INSERT INTO `resources` VALUES (249, 'image/edit.png', 'other', 'edit', 0, 1);
INSERT INTO `resources` VALUES (250, 'image/facebg_1.png', 'other', 'facebg_1', 0, 1);
INSERT INTO `resources` VALUES (251, 'image/facebook_login.jpg', 'other', 'facebook_login', 0, 1);
INSERT INTO `resources` VALUES (252, 'image/fresh_1.gif', 'other', 'fresh_1', 0, 1);
INSERT INTO `resources` VALUES (253, 'image/grouping_icons.png', 'other', 'grouping_icons', 0, 1);
INSERT INTO `resources` VALUES (254, 'image/grouping_icons_ie6.png', 'other', 'grouping_icons_ie6', 0, 1);
INSERT INTO `resources` VALUES (255, 'image/header.gif', 'other', 'header', 0, 1);
INSERT INTO `resources` VALUES (256, 'image/ico_star.png', 'other', 'ico_star', 0, 1);
INSERT INTO `resources` VALUES (257, 'image/icon_follow.png', 'other', 'icon_follow', 0, 1);
INSERT INTO `resources` VALUES (258, 'image/icon_soso.gif', 'other', 'icon_soso', 0, 1);
INSERT INTO `resources` VALUES (259, 'image/icon_tips.gif', 'other', 'icon_tips', 0, 1);
INSERT INTO `resources` VALUES (260, 'image/icon_tips.png', 'other', 'icon_tips', 0, 1);
INSERT INTO `resources` VALUES (261, 'image/icon_tools.png', 'other', 'icon_tools', 0, 1);
INSERT INTO `resources` VALUES (262, 'image/icon_user.png', 'other', 'icon_user', 0, 1);
INSERT INTO `resources` VALUES (263, 'image/icon_user_ie6.png', 'other', 'icon_user_ie6', 0, 1);
INSERT INTO `resources` VALUES (264, 'image/icon_video_play.png', 'other', 'icon_video_play', 0, 1);
INSERT INTO `resources` VALUES (265, 'image/icon_weibo_content.png', 'other', 'icon_weibo_content', 0, 1);
INSERT INTO `resources` VALUES (266, 'image/icons_card.png', 'other', 'icons_card', 0, 1);
INSERT INTO `resources` VALUES (267, 'image/icons_card_ie6.png', 'other', 'icons_card_ie6', 0, 1);
INSERT INTO `resources` VALUES (268, 'image/jzsy.gif', 'other', 'jzsy', 0, 1);
INSERT INTO `resources` VALUES (269, 'image/layer_bg.png', 'other', 'layer_bg', 0, 1);
INSERT INTO `resources` VALUES (270, 'image/loading1.gif', 'other', 'loading1', 0, 1);
INSERT INTO `resources` VALUES (271, 'image/loading11.gif', 'other', 'loading11', 0, 1);
INSERT INTO `resources` VALUES (272, 'image/loading2.gif', 'other', 'loading2', 0, 1);
INSERT INTO `resources` VALUES (273, 'image/loading3.gif', 'other', 'loading3', 0, 1);
INSERT INTO `resources` VALUES (274, 'image/loading4.gif', 'other', 'loading4', 0, 1);
INSERT INTO `resources` VALUES (275, 'image/loading_bar.gif', 'other', 'loading_bar', 0, 1);
INSERT INTO `resources` VALUES (276, 'image/login_borderbg.png', 'other', 'login_borderbg', 0, 1);
INSERT INTO `resources` VALUES (277, 'image/logo_question.png', 'other', 'logo_question', 0, 1);
INSERT INTO `resources` VALUES (278, 'image/logo_question_ie.png', 'other', 'logo_question_ie', 0, 1);
INSERT INTO `resources` VALUES (279, 'image/oly_adbg.jpg', 'other', 'oly_adbg', 0, 1);
INSERT INTO `resources` VALUES (280, 'image/oly_login.png', 'other', 'oly_login', 0, 1);
INSERT INTO `resources` VALUES (281, 'image/oly_login_slogan.png', 'other', 'oly_login_slogan', 0, 1);
INSERT INTO `resources` VALUES (282, 'image/publisher_icons.png', 'other', 'publisher_icons', 0, 1);
INSERT INTO `resources` VALUES (283, 'image/publisher_icons_ie6.png', 'other', 'publisher_icons_ie6', 0, 1);
INSERT INTO `resources` VALUES (284, 'image/qqlogo_2021.png', 'other', 'qqlogo_2021', 0, 1);
INSERT INTO `resources` VALUES (285, 'image/qqlogo_2021_ie.png', 'other', 'qqlogo_2021_ie', 0, 1);
INSERT INTO `resources` VALUES (286, 'image/reg_btn.png', 'other', 'reg_btn', 0, 1);
INSERT INTO `resources` VALUES (287, 'image/sharegame.png', 'other', 'sharegame', 0, 1);
INSERT INTO `resources` VALUES (288, 'image/shoptm.png', 'other', 'shoptm', 0, 1);
INSERT INTO `resources` VALUES (289, 'image/sprite.png', 'other', 'sprite', 0, 1);
INSERT INTO `resources` VALUES (290, 'image/state_fail.gif', 'other', 'state_fail', 0, 1);
INSERT INTO `resources` VALUES (291, 'image/state_fail.png', 'other', 'state_fail', 0, 1);
INSERT INTO `resources` VALUES (292, 'image/tip_bg.png', 'other', 'tip_bg', 0, 1);
INSERT INTO `resources` VALUES (293, 'image/ui_items.png', 'other', 'ui_items', 0, 1);
INSERT INTO `resources` VALUES (294, 'image/ui_items_ie6.png', 'other', 'ui_items_ie6', 0, 1);
INSERT INTO `resources` VALUES (295, 'image/upload_status_mark.png', 'other', 'upload_status_mark', 0, 1);
INSERT INTO `resources` VALUES (296, 'image/vip_logo.gif', 'other', 'vip_logo', 0, 1);
INSERT INTO `resources` VALUES (297, 'image/wb_logo.gif', 'other', 'wb_logo', 0, 1);
INSERT INTO `resources` VALUES (298, 'image/wb_logo.png', 'other', 'wb_logo', 0, 1);
INSERT INTO `resources` VALUES (299, 'image/wb_logo16_a.png', 'other', 'wb_logo16_a', 0, 1);
INSERT INTO `resources` VALUES (300, 'image/wb_xline_s1.gif', 'other', 'wb_xline_s1', 0, 1);
INSERT INTO `resources` VALUES (301, 'image/wb_xline_s1.png', 'other', 'wb_xline_s1', 0, 1);
INSERT INTO `resources` VALUES (302, 'image/wb_xline_s2.png', 'other', 'wb_xline_s2', 0, 1);
INSERT INTO `resources` VALUES (303, 'image/wea_arrow.gif', 'other', 'wea_arrow', 0, 1);
INSERT INTO `resources` VALUES (304, 'image/weimi_oauth_btn.png', 'other', 'weimi_oauth_btn', 0, 1);
INSERT INTO `resources` VALUES (305, 'image/wemeet_cover.png', 'other', 'wemeet_cover', 0, 1);
INSERT INTO `resources` VALUES (306, 'image/wemeet_icon.png', 'other', 'wemeet_icon', 0, 1);
INSERT INTO `resources` VALUES (307, 'image/wemeet_none_avatar.png', 'other', 'wemeet_none_avatar', 0, 1);
INSERT INTO `resources` VALUES (308, 'picture/sq1.1.gif', 'priv', '超Q 1级', 1, 1);
INSERT INTO `resources` VALUES (309, 'picture/sq1.2.gif', 'priv', '超Q 2级', 2, 1);
INSERT INTO `resources` VALUES (310, 'picture/sq1.3.gif', 'priv', '超Q 3级', 3, 1);
INSERT INTO `resources` VALUES (311, 'picture/sq1.4.gif', 'priv', '超Q 4级', 4, 1);
INSERT INTO `resources` VALUES (312, 'picture/sq1.5.gif', 'priv', '超Q 5级', 5, 1);
INSERT INTO `resources` VALUES (313, 'picture/sq1.6.gif', 'priv', '超Q 6级', 6, 1);
INSERT INTO `resources` VALUES (314, 'picture/sq1.7.gif', 'priv', '超Q 7级', 7, 1);
INSERT INTO `resources` VALUES (315, 'picture/sq1.8.gif', 'priv', '超Q 8级', 8, 1);
INSERT INTO `resources` VALUES (316, 'picture/lz2.1.gif', 'priv', '蓝钻 1级', 9, 1);
INSERT INTO `resources` VALUES (317, 'picture/lz2.2.gif', 'priv', '蓝钻 2级', 10, 1);
INSERT INTO `resources` VALUES (318, 'picture/lz2.3.gif', 'priv', '蓝钻 3级', 11, 1);
INSERT INTO `resources` VALUES (319, 'picture/lz2.4.gif', 'priv', '蓝钻 4级', 12, 1);
INSERT INTO `resources` VALUES (320, 'picture/lz2.5.gif', 'priv', '蓝钻 5级', 13, 1);
INSERT INTO `resources` VALUES (321, 'picture/lz2.6.gif', 'priv', '蓝钻 6级', 14, 1);
INSERT INTO `resources` VALUES (322, 'picture/lz2.7.gif', 'priv', '蓝钻 7级', 15, 1);
INSERT INTO `resources` VALUES (323, 'picture/lz2.8.gif', 'priv', '蓝钻 8级', 16, 1);
INSERT INTO `resources` VALUES (324, 'picture/v1.gif', 'other', 'v1', 0, 1);
INSERT INTO `resources` VALUES (325, 'picture/v10.gif', 'other', 'v10', 0, 1);
INSERT INTO `resources` VALUES (326, 'picture/v13.gif', 'other', 'v13', 0, 1);
INSERT INTO `resources` VALUES (327, 'picture/v16.gif', 'other', 'v16', 0, 1);
INSERT INTO `resources` VALUES (328, 'picture/v18.gif', 'other', 'v18', 0, 1);
INSERT INTO `resources` VALUES (329, 'picture/v36.gif', 'other', 'v36', 0, 1);
INSERT INTO `resources` VALUES (330, 'picture/v6.gif', 'other', 'v6', 0, 1);
INSERT INTO `resources` VALUES (331, 'picture/v9.gif', 'other', 'v9', 0, 1);
INSERT INTO `resources` VALUES (332, 'image/05.gif', 'other', '05', 0, 1);
INSERT INTO `resources` VALUES (333, 'image/1.gif', 'other', '1', 0, 1);
INSERT INTO `resources` VALUES (334, 'image/blog.gif', 'other', 'blog', 0, 1);
INSERT INTO `resources` VALUES (335, 'image/home.gif', 'other', 'home', 0, 1);
INSERT INTO `resources` VALUES (336, 'image/id.gif', 'other', 'id', 0, 1);
INSERT INTO `resources` VALUES (337, 'image/vipqq.jpg', 'other', 'vipqq', 0, 1);
INSERT INTO `resources` VALUES (338, 'picture/bpm_1.gif', 'other', 'bpm_1', 0, 1);
INSERT INTO `resources` VALUES (339, 'picture/marksix_1.gif', 'other', 'marksix_1', 0, 1);
INSERT INTO `resources` VALUES (340, 'picture/qs_05.gif', 'other', 'qs_05', 0, 1);
INSERT INTO `resources` VALUES (341, 'picture/qs_10.gif', 'other', 'qs_10', 0, 1);
INSERT INTO `resources` VALUES (342, 'picture/qs_11.gif', 'other', 'qs_11', 0, 1);
INSERT INTO `resources` VALUES (343, 'picture/qs_12.gif', 'other', 'qs_12', 0, 1);
INSERT INTO `resources` VALUES (344, 'picture/qs_13.gif', 'other', 'qs_13', 0, 1);
INSERT INTO `resources` VALUES (345, 'picture/chuping.jpg', 'other', 'chuping', 0, 1);
INSERT INTO `resources` VALUES (346, 'picture/home_1_1.gif', 'other', 'home_1_1', 0, 1);
INSERT INTO `resources` VALUES (347, 'picture/noble_1_1.gif', 'other', 'noble_1_1', 0, 1);
INSERT INTO `resources` VALUES (348, 'picture/noble_2_1.gif', 'other', 'noble_2_1', 0, 1);
INSERT INTO `resources` VALUES (349, 'image/youxi.gif', 'other', 'youxi', 0, 1);
INSERT INTO `resources` VALUES (350, 'picture/m_s_1.gif', 'other', 'm_s_1', 0, 1);
INSERT INTO `resources` VALUES (351, 'picture/m_s_10.gif', 'other', 'm_s_10', 0, 1);
INSERT INTO `resources` VALUES (352, 'picture/m_s_11.gif', 'other', 'm_s_11', 0, 1);
INSERT INTO `resources` VALUES (353, 'picture/m_s_12.gif', 'other', 'm_s_12', 0, 1);
INSERT INTO `resources` VALUES (354, 'picture/m_s_13.gif', 'other', 'm_s_13', 0, 1);
INSERT INTO `resources` VALUES (355, 'picture/m_s_14.gif', 'other', 'm_s_14', 0, 1);
INSERT INTO `resources` VALUES (356, 'picture/m_s_15.gif', 'other', 'm_s_15', 0, 1);
INSERT INTO `resources` VALUES (357, 'picture/m_s_16.gif', 'other', 'm_s_16', 0, 1);
INSERT INTO `resources` VALUES (358, 'picture/m_s_17.gif', 'other', 'm_s_17', 0, 1);
INSERT INTO `resources` VALUES (359, 'picture/m_s_18.gif', 'other', 'm_s_18', 0, 1);
INSERT INTO `resources` VALUES (360, 'picture/m_s_19.gif', 'other', 'm_s_19', 0, 1);
INSERT INTO `resources` VALUES (361, 'picture/m_s_2.gif', 'other', 'm_s_2', 0, 1);
INSERT INTO `resources` VALUES (362, 'picture/m_s_20.gif', 'other', 'm_s_20', 0, 1);
INSERT INTO `resources` VALUES (363, 'picture/m_s_21.gif', 'other', 'm_s_21', 0, 1);
INSERT INTO `resources` VALUES (364, 'picture/m_s_22.gif', 'other', 'm_s_22', 0, 1);
INSERT INTO `resources` VALUES (365, 'picture/m_s_23.gif', 'other', 'm_s_23', 0, 1);
INSERT INTO `resources` VALUES (366, 'picture/m_s_24.gif', 'other', 'm_s_24', 0, 1);
INSERT INTO `resources` VALUES (367, 'picture/m_s_25.gif', 'other', 'm_s_25', 0, 1);
INSERT INTO `resources` VALUES (368, 'picture/m_s_26.gif', 'other', 'm_s_26', 0, 1);
INSERT INTO `resources` VALUES (369, 'picture/m_s_27.gif', 'other', 'm_s_27', 0, 1);
INSERT INTO `resources` VALUES (370, 'picture/m_s_28.gif', 'other', 'm_s_28', 0, 1);
INSERT INTO `resources` VALUES (371, 'picture/m_s_29.gif', 'other', 'm_s_29', 0, 1);
INSERT INTO `resources` VALUES (372, 'picture/m_s_3.gif', 'other', 'm_s_3', 0, 1);
INSERT INTO `resources` VALUES (373, 'picture/m_s_30.gif', 'other', 'm_s_30', 0, 1);
INSERT INTO `resources` VALUES (374, 'picture/m_s_31.gif', 'other', 'm_s_31', 0, 1);
INSERT INTO `resources` VALUES (375, 'picture/m_s_32.gif', 'other', 'm_s_32', 0, 1);
INSERT INTO `resources` VALUES (376, 'picture/m_s_33.gif', 'other', 'm_s_33', 0, 1);
INSERT INTO `resources` VALUES (377, 'picture/m_s_34.gif', 'other', 'm_s_34', 0, 1);
INSERT INTO `resources` VALUES (378, 'picture/m_s_35.gif', 'other', 'm_s_35', 0, 1);
INSERT INTO `resources` VALUES (379, 'picture/m_s_36.gif', 'other', 'm_s_36', 0, 1);
INSERT INTO `resources` VALUES (380, 'picture/m_s_37.gif', 'other', 'm_s_37', 0, 1);
INSERT INTO `resources` VALUES (381, 'picture/m_s_38.gif', 'other', 'm_s_38', 0, 1);
INSERT INTO `resources` VALUES (382, 'picture/m_s_39.gif', 'other', 'm_s_39', 0, 1);
INSERT INTO `resources` VALUES (383, 'picture/m_s_4.gif', 'other', 'm_s_4', 0, 1);
INSERT INTO `resources` VALUES (384, 'picture/m_s_40.gif', 'other', 'm_s_40', 0, 1);
INSERT INTO `resources` VALUES (385, 'picture/m_s_5.gif', 'other', 'm_s_5', 0, 1);
INSERT INTO `resources` VALUES (386, 'picture/m_s_6.gif', 'other', 'm_s_6', 0, 1);
INSERT INTO `resources` VALUES (387, 'picture/m_s_7.gif', 'other', 'm_s_7', 0, 1);
INSERT INTO `resources` VALUES (388, 'picture/m_s_8.gif', 'other', 'm_s_8', 0, 1);
INSERT INTO `resources` VALUES (389, 'picture/m_s_9.gif', 'other', 'm_s_9', 0, 1);
INSERT INTO `resources` VALUES (390, 'picture/hot.gif', 'other', 'hot', 0, 1);
INSERT INTO `resources` VALUES (391, 'picture/home_2_2.gif', 'other', 'home_2_2', 0, 1);
INSERT INTO `resources` VALUES (392, 'picture/noble_1_2.gif', 'other', 'noble_1_2', 0, 1);
INSERT INTO `resources` VALUES (393, 'picture/noble_1_3.gif', 'other', 'noble_1_3', 0, 1);
INSERT INTO `resources` VALUES (394, 'picture/noble_1_4.gif', 'other', 'noble_1_4', 0, 1);
INSERT INTO `resources` VALUES (395, 'picture/noble_1_5.gif', 'other', 'noble_1_5', 0, 1);
INSERT INTO `resources` VALUES (396, 'picture/noble_1_6.gif', 'other', 'noble_1_6', 0, 1);
INSERT INTO `resources` VALUES (397, 'picture/noble_1_7.gif', 'other', 'noble_1_7', 0, 1);
INSERT INTO `resources` VALUES (398, 'picture/noble_1_8.gif', 'other', 'noble_1_8', 0, 1);
INSERT INTO `resources` VALUES (399, 'picture/noble_2_2.gif', 'other', 'noble_2_2', 0, 1);
INSERT INTO `resources` VALUES (400, 'picture/noble_2_3.gif', 'other', 'noble_2_3', 0, 1);
INSERT INTO `resources` VALUES (401, 'picture/noble_2_4.gif', 'other', 'noble_2_4', 0, 1);
INSERT INTO `resources` VALUES (402, 'picture/noble_2_5.gif', 'other', 'noble_2_5', 0, 1);
INSERT INTO `resources` VALUES (403, 'picture/noble_2_6.gif', 'other', 'noble_2_6', 0, 1);
INSERT INTO `resources` VALUES (404, 'picture/noble_2_7.gif', 'other', 'noble_2_7', 0, 1);
INSERT INTO `resources` VALUES (405, 'picture/noble_2_8.gif', 'other', 'noble_2_8', 0, 1);
INSERT INTO `resources` VALUES (406, 'image/home1.png', 'other', 'home1', 0, 1);
INSERT INTO `resources` VALUES (407, 'picture/home_1_10.gif', 'other', 'home_1_10', 0, 1);
INSERT INTO `resources` VALUES (408, 'picture/home_1_11.gif', 'other', 'home_1_11', 0, 1);
INSERT INTO `resources` VALUES (409, 'picture/home_1_12.gif', 'other', 'home_1_12', 0, 1);
INSERT INTO `resources` VALUES (410, 'picture/home_1_13.gif', 'other', 'home_1_13', 0, 1);
INSERT INTO `resources` VALUES (411, 'picture/home_1_14.gif', 'other', 'home_1_14', 0, 1);
INSERT INTO `resources` VALUES (412, 'picture/home_1_15.gif', 'other', 'home_1_15', 0, 1);
INSERT INTO `resources` VALUES (413, 'picture/home_1_16.gif', 'other', 'home_1_16', 0, 1);
INSERT INTO `resources` VALUES (414, 'picture/home_1_17.gif', 'other', 'home_1_17', 0, 1);
INSERT INTO `resources` VALUES (415, 'picture/home_1_2.gif', 'other', 'home_1_2', 0, 1);
INSERT INTO `resources` VALUES (416, 'picture/home_1_3.gif', 'other', 'home_1_3', 0, 1);
INSERT INTO `resources` VALUES (417, 'picture/home_1_4.gif', 'other', 'home_1_4', 0, 1);
INSERT INTO `resources` VALUES (418, 'picture/home_1_5.gif', 'other', 'home_1_5', 0, 1);
INSERT INTO `resources` VALUES (419, 'picture/home_1_6.gif', 'other', 'home_1_6', 0, 1);
INSERT INTO `resources` VALUES (420, 'picture/home_1_7.gif', 'other', 'home_1_7', 0, 1);
INSERT INTO `resources` VALUES (421, 'picture/home_1_8.gif', 'other', 'home_1_8', 0, 1);
INSERT INTO `resources` VALUES (422, 'picture/home_1_9.gif', 'other', 'home_1_9', 0, 1);
INSERT INTO `resources` VALUES (423, 'picture/home_2_1.gif', 'other', 'home_2_1', 0, 1);
INSERT INTO `resources` VALUES (424, 'picture/home_2_10.gif', 'other', 'home_2_10', 0, 1);
INSERT INTO `resources` VALUES (425, 'picture/home_2_11.gif', 'other', 'home_2_11', 0, 1);
INSERT INTO `resources` VALUES (426, 'picture/home_2_12.gif', 'other', 'home_2_12', 0, 1);
INSERT INTO `resources` VALUES (427, 'picture/home_2_13.gif', 'other', 'home_2_13', 0, 1);
INSERT INTO `resources` VALUES (428, 'picture/home_2_14.gif', 'other', 'home_2_14', 0, 1);
INSERT INTO `resources` VALUES (429, 'picture/home_2_15.gif', 'other', 'home_2_15', 0, 1);
INSERT INTO `resources` VALUES (430, 'picture/home_2_16.gif', 'other', 'home_2_16', 0, 1);
INSERT INTO `resources` VALUES (431, 'picture/home_2_17.gif', 'other', 'home_2_17', 0, 1);
INSERT INTO `resources` VALUES (432, 'picture/home_2_3.gif', 'other', 'home_2_3', 0, 1);
INSERT INTO `resources` VALUES (433, 'picture/home_2_4.gif', 'other', 'home_2_4', 0, 1);
INSERT INTO `resources` VALUES (434, 'picture/home_2_5.gif', 'other', 'home_2_5', 0, 1);
INSERT INTO `resources` VALUES (435, 'picture/home_2_6.gif', 'other', 'home_2_6', 0, 1);
INSERT INTO `resources` VALUES (436, 'picture/home_2_7.gif', 'other', 'home_2_7', 0, 1);
INSERT INTO `resources` VALUES (437, 'picture/home_2_8.gif', 'other', 'home_2_8', 0, 1);
INSERT INTO `resources` VALUES (438, 'picture/home_2_9.gif', 'other', 'home_2_9', 0, 1);
INSERT INTO `resources` VALUES (439, 'picture/101_u.gif', 'other', '101_u', 0, 1);
INSERT INTO `resources` VALUES (440, 'picture/202_u.jpg', 'other', '202_u', 0, 1);
INSERT INTO `resources` VALUES (441, 'picture/301_u.gif', 'other', '301_u', 0, 1);
INSERT INTO `resources` VALUES (442, 'picture/401_u.gif', 'other', '401_u', 0, 1);
INSERT INTO `resources` VALUES (443, 'picture/501_u.gif', 'other', '501_u', 0, 1);
INSERT INTO `resources` VALUES (444, 'picture/jindu.jpg', 'other', 'jindu', 0, 1);
INSERT INTO `resources` VALUES (445, 'picture/home_1_18.gif', 'other', 'home_1_18', 0, 1);
INSERT INTO `resources` VALUES (446, 'picture/home_1_19.gif', 'other', 'home_1_19', 0, 1);
INSERT INTO `resources` VALUES (447, 'picture/home_1_20.gif', 'other', 'home_1_20', 0, 1);
INSERT INTO `resources` VALUES (448, 'picture/home_1_21.gif', 'other', 'home_1_21', 0, 1);
INSERT INTO `resources` VALUES (449, 'picture/home_1_22.gif', 'other', 'home_1_22', 0, 1);
INSERT INTO `resources` VALUES (450, 'picture/home_1_23.gif', 'other', 'home_1_23', 0, 1);
INSERT INTO `resources` VALUES (451, 'picture/home_1_24.gif', 'other', 'home_1_24', 0, 1);
INSERT INTO `resources` VALUES (452, 'picture/home_1_25.gif', 'other', 'home_1_25', 0, 1);
INSERT INTO `resources` VALUES (453, 'picture/home_1_26.gif', 'other', 'home_1_26', 0, 1);
INSERT INTO `resources` VALUES (454, 'picture/home_1_27.gif', 'other', 'home_1_27', 0, 1);
INSERT INTO `resources` VALUES (455, 'picture/home_1_28.gif', 'other', 'home_1_28', 0, 1);
INSERT INTO `resources` VALUES (456, 'picture/home_1_29.gif', 'other', 'home_1_29', 0, 1);
INSERT INTO `resources` VALUES (457, 'picture/home_1_30.gif', 'other', 'home_1_30', 0, 1);
INSERT INTO `resources` VALUES (458, 'picture/home_1_31.gif', 'other', 'home_1_31', 0, 1);
INSERT INTO `resources` VALUES (459, 'picture/home_1_32.gif', 'other', 'home_1_32', 0, 1);
INSERT INTO `resources` VALUES (460, 'picture/home_1_33.gif', 'other', 'home_1_33', 0, 1);
INSERT INTO `resources` VALUES (461, 'picture/home_1_34.gif', 'other', 'home_1_34', 0, 1);
INSERT INTO `resources` VALUES (462, 'picture/home_1_35.gif', 'other', 'home_1_35', 0, 1);
INSERT INTO `resources` VALUES (463, 'picture/home_1_36.gif', 'other', 'home_1_36', 0, 1);
INSERT INTO `resources` VALUES (464, 'picture/home_1_37.gif', 'other', 'home_1_37', 0, 1);
INSERT INTO `resources` VALUES (465, 'picture/home_1_38.gif', 'other', 'home_1_38', 0, 1);
INSERT INTO `resources` VALUES (466, 'picture/home_1_39.gif', 'other', 'home_1_39', 0, 1);
INSERT INTO `resources` VALUES (467, 'picture/home_1_40.gif', 'other', 'home_1_40', 0, 1);
INSERT INTO `resources` VALUES (468, 'picture/home_1_41.gif', 'other', 'home_1_41', 0, 1);
INSERT INTO `resources` VALUES (469, 'picture/home_1_42.gif', 'other', 'home_1_42', 0, 1);
INSERT INTO `resources` VALUES (470, 'picture/home_1_43.gif', 'other', 'home_1_43', 0, 1);
INSERT INTO `resources` VALUES (471, 'picture/home_1_44.gif', 'other', 'home_1_44', 0, 1);
INSERT INTO `resources` VALUES (472, 'picture/home_1_45.gif', 'other', 'home_1_45', 0, 1);
INSERT INTO `resources` VALUES (473, 'picture/home_1_46.gif', 'other', 'home_1_46', 0, 1);
INSERT INTO `resources` VALUES (474, 'picture/home_1_47.gif', 'other', 'home_1_47', 0, 1);
INSERT INTO `resources` VALUES (475, 'picture/home_1_48.gif', 'other', 'home_1_48', 0, 1);
INSERT INTO `resources` VALUES (476, 'picture/home_1_49.gif', 'other', 'home_1_49', 0, 1);
INSERT INTO `resources` VALUES (477, 'picture/home_1_50.gif', 'other', 'home_1_50', 0, 1);
INSERT INTO `resources` VALUES (478, 'picture/home_2_18.gif', 'other', 'home_2_18', 0, 1);
INSERT INTO `resources` VALUES (479, 'picture/home_2_19.gif', 'other', 'home_2_19', 0, 1);
INSERT INTO `resources` VALUES (480, 'picture/home_2_20.gif', 'other', 'home_2_20', 0, 1);
INSERT INTO `resources` VALUES (481, 'picture/home_2_21.gif', 'other', 'home_2_21', 0, 1);
INSERT INTO `resources` VALUES (482, 'picture/home_2_22.gif', 'other', 'home_2_22', 0, 1);
INSERT INTO `resources` VALUES (483, 'picture/home_2_23.gif', 'other', 'home_2_23', 0, 1);
INSERT INTO `resources` VALUES (484, 'picture/home_2_24.gif', 'other', 'home_2_24', 0, 1);
INSERT INTO `resources` VALUES (485, 'picture/home_2_25.gif', 'other', 'home_2_25', 0, 1);
INSERT INTO `resources` VALUES (486, 'picture/home_2_26.gif', 'other', 'home_2_26', 0, 1);
INSERT INTO `resources` VALUES (487, 'picture/home_2_27.gif', 'other', 'home_2_27', 0, 1);
INSERT INTO `resources` VALUES (488, 'picture/home_2_28.gif', 'other', 'home_2_28', 0, 1);
INSERT INTO `resources` VALUES (489, 'picture/home_2_29.gif', 'other', 'home_2_29', 0, 1);
INSERT INTO `resources` VALUES (490, 'picture/home_2_30.gif', 'other', 'home_2_30', 0, 1);
INSERT INTO `resources` VALUES (491, 'picture/home_2_31.gif', 'other', 'home_2_31', 0, 1);
INSERT INTO `resources` VALUES (492, 'picture/home_2_32.gif', 'other', 'home_2_32', 0, 1);
INSERT INTO `resources` VALUES (493, 'picture/home_2_33.gif', 'other', 'home_2_33', 0, 1);
INSERT INTO `resources` VALUES (494, 'picture/home_2_34.gif', 'other', 'home_2_34', 0, 1);
INSERT INTO `resources` VALUES (495, 'picture/home_2_35.gif', 'other', 'home_2_35', 0, 1);
INSERT INTO `resources` VALUES (496, 'picture/home_2_36.gif', 'other', 'home_2_36', 0, 1);
INSERT INTO `resources` VALUES (497, 'picture/home_2_37.gif', 'other', 'home_2_37', 0, 1);
INSERT INTO `resources` VALUES (498, 'picture/home_2_38.gif', 'other', 'home_2_38', 0, 1);
INSERT INTO `resources` VALUES (499, 'picture/home_2_39.gif', 'other', 'home_2_39', 0, 1);
INSERT INTO `resources` VALUES (500, 'picture/home_2_40.gif', 'other', 'home_2_40', 0, 1);
INSERT INTO `resources` VALUES (501, 'picture/home_2_41.gif', 'other', 'home_2_41', 0, 1);
INSERT INTO `resources` VALUES (502, 'picture/home_2_42.gif', 'other', 'home_2_42', 0, 1);
INSERT INTO `resources` VALUES (503, 'picture/home_2_43.gif', 'other', 'home_2_43', 0, 1);
INSERT INTO `resources` VALUES (504, 'picture/home_2_44.gif', 'other', 'home_2_44', 0, 1);
INSERT INTO `resources` VALUES (505, 'picture/home_2_45.gif', 'other', 'home_2_45', 0, 1);
INSERT INTO `resources` VALUES (506, 'picture/home_2_46.gif', 'other', 'home_2_46', 0, 1);
INSERT INTO `resources` VALUES (507, 'picture/home_2_47.gif', 'other', 'home_2_47', 0, 1);
INSERT INTO `resources` VALUES (508, 'picture/home_2_48.gif', 'other', 'home_2_48', 0, 1);
INSERT INTO `resources` VALUES (509, 'picture/home_2_49.gif', 'other', 'home_2_49', 0, 1);
INSERT INTO `resources` VALUES (510, 'picture/home_2_50.gif', 'other', 'home_2_50', 0, 1);
INSERT INTO `resources` VALUES (511, 'image/jiazu.gif', 'other', 'jiazu', 0, 1);
INSERT INTO `resources` VALUES (512, 'picture/mon00.gif', 'other', 'mon00', 0, 1);
INSERT INTO `resources` VALUES (513, 'picture/mon01.gif', 'other', 'mon01', 0, 1);
INSERT INTO `resources` VALUES (514, 'picture/mon02.gif', 'other', 'mon02', 0, 1);
INSERT INTO `resources` VALUES (515, 'picture/mon03.png', 'other', 'mon03', 0, 1);
INSERT INTO `resources` VALUES (516, 'image/noble_2_1.gif', 'other', 'noble_2_1', 0, 1);
INSERT INTO `resources` VALUES (517, 'image/noble_2_2.gif', 'other', 'noble_2_2', 0, 1);
INSERT INTO `resources` VALUES (518, 'picture/1f4d3d7503c6199d80bd95bdbaa05bbf.gif', 'other', '1f4d3d7503c6199d80bd95bdbaa05b', 0, 1);
INSERT INTO `resources` VALUES (519, 'picture/4d11f86d57354f0b340c2f287c140c70.gif', 'other', '4d11f86d57354f0b340c2f287c140c', 0, 1);

-- ----------------------------
-- Table structure for role_permissions
-- ----------------------------
DROP TABLE IF EXISTS `role_permissions`;
CREATE TABLE `role_permissions`  (
  `role_id` bigint UNSIGNED NOT NULL,
  `permission_id` bigint UNSIGNED NOT NULL,
  PRIMARY KEY (`role_id`, `permission_id`) USING BTREE,
  INDEX `fk_role_permissions_permission`(`permission_id` ASC) USING BTREE,
  CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of role_permissions
-- ----------------------------
INSERT INTO `role_permissions` VALUES (1, 1);
INSERT INTO `role_permissions` VALUES (2, 1);
INSERT INTO `role_permissions` VALUES (3, 1);
INSERT INTO `role_permissions` VALUES (1, 2);
INSERT INTO `role_permissions` VALUES (2, 2);
INSERT INTO `role_permissions` VALUES (1, 3);
INSERT INTO `role_permissions` VALUES (2, 3);
INSERT INTO `role_permissions` VALUES (1, 4);
INSERT INTO `role_permissions` VALUES (2, 4);
INSERT INTO `role_permissions` VALUES (3, 4);
INSERT INTO `role_permissions` VALUES (1, 5);
INSERT INTO `role_permissions` VALUES (2, 5);
INSERT INTO `role_permissions` VALUES (1, 6);
INSERT INTO `role_permissions` VALUES (1, 7);
INSERT INTO `role_permissions` VALUES (2, 7);
INSERT INTO `role_permissions` VALUES (1, 8);
INSERT INTO `role_permissions` VALUES (2, 8);

-- ----------------------------
-- Table structure for roles
-- ----------------------------
DROP TABLE IF EXISTS `roles`;
CREATE TABLE `roles`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `code` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_roles_name`(`name` ASC) USING BTREE,
  UNIQUE INDEX `idx_roles_code`(`code` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 5 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of roles
-- ----------------------------
INSERT INTO `roles` VALUES (1, '超级管理员', 'super_admin', '拥有全部权限');
INSERT INTO `roles` VALUES (2, '管理员', 'admin', '社区日常管理');
INSERT INTO `roles` VALUES (3, '版主', 'moderator', '管理帖子');
INSERT INTO `roles` VALUES (4, '普通会员', 'member', '注册用户默认角色');

-- ----------------------------
-- Table structure for settings
-- ----------------------------
DROP TABLE IF EXISTS `settings`;
CREATE TABLE `settings`  (
  `key` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `value` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  PRIMARY KEY (`key`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of settings
-- ----------------------------
INSERT INTO `settings` VALUES ('ttou_user_id', '10000');

-- ----------------------------
-- Table structure for sign_ins
-- ----------------------------
DROP TABLE IF EXISTS `sign_ins`;
CREATE TABLE `sign_ins`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sign_date` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `consec` bigint NULL DEFAULT NULL,
  `reward` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_user_date`(`user_id` ASC, `sign_date` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 7 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of sign_ins
-- ----------------------------
INSERT INTO `sign_ins` VALUES (2, 10000, '2026-08-30', 1, 11, '2026-08-30 10:28:02.832');
INSERT INTO `sign_ins` VALUES (3, 10007, '2026-08-30', 1, 11, '2026-08-30 10:37:00.400');
INSERT INTO `sign_ins` VALUES (4, 10007, '2026-09-04', 1, 11, '2026-09-04 10:39:36.334');
INSERT INTO `sign_ins` VALUES (5, 10001, '2026-09-05', 1, 11, '2026-09-05 16:48:01.298');
INSERT INTO `sign_ins` VALUES (6, 10000, '2026-09-05', 1, 11, '2026-09-05 20:05:29.368');

-- ----------------------------
-- Table structure for space_messages
-- ----------------------------
DROP TABLE IF EXISTS `space_messages`;
CREATE TABLE `space_messages`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `to_user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `from_user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `content` varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_space_messages_to_user_id`(`to_user_id` ASC) USING BTREE,
  INDEX `idx_space_messages_from_user_id`(`from_user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of space_messages
-- ----------------------------

-- ----------------------------
-- Table structure for spaces
-- ----------------------------
DROP TABLE IF EXISTS `spaces`;
CREATE TABLE `spaces`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `signature` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `intro` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_spaces_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of spaces
-- ----------------------------
INSERT INTO `spaces` VALUES (1, 10007, '夜凌云的空间', '签名', '说明', 1, '2026-08-30 22:40:14.629', '2026-08-30 22:40:14.629');
INSERT INTO `spaces` VALUES (2, 10001, '我的新空间', '我的新空间', '我的新空间', 1, '2026-09-05 17:26:50.047', '2026-09-05 17:26:50.047');

-- ----------------------------
-- Table structure for thread_favorites
-- ----------------------------
DROP TABLE IF EXISTS `thread_favorites`;
CREATE TABLE `thread_favorites`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `thread_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_fav_user`(`user_id` ASC, `thread_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of thread_favorites
-- ----------------------------
INSERT INTO `thread_favorites` VALUES (1, 10001, 10001, '2026-09-04 11:05:37.361');
INSERT INTO `thread_favorites` VALUES (3, 10001, 10013, '2026-09-05 15:54:14.829');

-- ----------------------------
-- Table structure for thread_flowers
-- ----------------------------
DROP TABLE IF EXISTS `thread_flowers`;
CREATE TABLE `thread_flowers`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `thread_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sender_id` bigint UNSIGNED NULL DEFAULT NULL,
  `flower` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `count` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_thread_flowers_thread_id`(`thread_id` ASC) USING BTREE,
  INDEX `idx_thread_flowers_sender_id`(`sender_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of thread_flowers
-- ----------------------------
INSERT INTO `thread_flowers` VALUES (1, 10038, 10000, '玫瑰花', 2, '2026-09-05 16:07:55.631');

-- ----------------------------
-- Table structure for thread_gifts
-- ----------------------------
DROP TABLE IF EXISTS `thread_gifts`;
CREATE TABLE `thread_gifts`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `thread_id` bigint UNSIGNED NULL DEFAULT NULL,
  `sender_id` bigint UNSIGNED NULL DEFAULT NULL,
  `coins` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_thread_gifts_thread_id`(`thread_id` ASC) USING BTREE,
  INDEX `idx_thread_gifts_sender_id`(`sender_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of thread_gifts
-- ----------------------------
INSERT INTO `thread_gifts` VALUES (1, 10037, 10001, 100, '2026-09-05 15:46:30.886');
INSERT INTO `thread_gifts` VALUES (2, 10038, 10000, 100, '2026-09-05 20:05:12.663');

-- ----------------------------
-- Table structure for thread_votes
-- ----------------------------
DROP TABLE IF EXISTS `thread_votes`;
CREATE TABLE `thread_votes`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `thread_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `value` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_tv`(`thread_id` ASC, `user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 4 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of thread_votes
-- ----------------------------
INSERT INTO `thread_votes` VALUES (1, 10038, 10000, -1, '2026-09-05 15:45:39.366');
INSERT INTO `thread_votes` VALUES (2, 10037, 10001, 1, '2026-09-05 15:46:27.718');
INSERT INTO `thread_votes` VALUES (3, 10038, 10001, 1, '2026-09-05 16:11:46.538');

-- ----------------------------
-- Table structure for threads
-- ----------------------------
DROP TABLE IF EXISTS `threads`;
CREATE TABLE `threads`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `board_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `is_top` bigint NULL DEFAULT 0,
  `is_fine` bigint NULL DEFAULT 0,
  `view_count` bigint NULL DEFAULT 0,
  `reply_count` bigint NULL DEFAULT 0,
  `status` bigint NULL DEFAULT 1,
  `last_reply_at` datetime(3) NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `like_count` bigint NULL DEFAULT 0,
  `dislike_count` bigint NULL DEFAULT 0,
  `share_count` bigint NULL DEFAULT 0,
  `gift_total` bigint NULL DEFAULT 0,
  `flower_count` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_threads_board_id`(`board_id` ASC) USING BTREE,
  INDEX `idx_threads_user_id`(`user_id` ASC) USING BTREE,
  CONSTRAINT `fk_threads_board` FOREIGN KEY (`board_id`) REFERENCES `boards` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `fk_threads_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB AUTO_INCREMENT = 10039 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of threads
-- ----------------------------
INSERT INTO `threads` VALUES (10000, 7, 10000, '【新手大全实用手册】', '欢迎加入社区。\n幸甚曾拥有三猪，拥有曾几何时青春澎湃的你们。有人在这寻得单纯的友情、青涩的爱情，亦有人寻得逛街的水友，皆由缘起。\n\n【论坛经验等级对照表】发帖+10经验，回帖+5经验，签到+20经验。\n【日常签到】每天签到可得金币，连续7天有惊喜！\n【新人礼包】注册即送100金币，签到还能翻倍哦。\n\n　　　　　　　　家园社区欢迎你！', 1, 1, 1163, 4, 1, '2026-08-30 10:39:17.831', '2026-08-30 10:07:01.678', '2026-08-30 10:39:17.831', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10001, 5, 10004, '又一个多月没来了', '工作太忙，好不容易摸鱼上来看看，家人们都还好吗？还记得当年的抢车位和好友买卖吗，感慨啊……', 0, 0, 38, 5, 1, '2026-08-30 10:28:38.165', '2026-08-30 10:07:01.707', '2026-08-30 10:28:38.165', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10002, 5, 10001, '【唱响论坛业务】你点我唱音乐业务正式开业', '规则：点歌请按格式回复——【点歌】歌曲名+赠言。楼主翻唱后@你。\n今日曲目单：孤勇者、晴天、七里香、突然的自我。', 0, 1, 234, 3, 1, '2026-08-29 13:07:01.677', '2026-08-30 10:07:01.741', '2026-08-30 10:07:01.758', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10003, 6, 10002, '漂亮的姐妹头', '分享几张自己画的头像，姐妹们喜欢可以自取，记得回帖告诉我用了哪张哦~', 0, 0, 158, 2, 1, '2026-08-29 10:07:01.677', '2026-08-30 10:07:01.768', '2026-08-30 10:07:01.781', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10004, 12, 10001, '睡觉睡觉', '夜深了，福建的老乡们都在吗？报个到一起聊聊天。', 0, 0, 65, 2, 1, '2026-08-29 10:07:01.677', '2026-08-30 10:07:01.790', '2026-08-30 10:07:01.804', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10005, 13, 10005, '孤独的人无所谓', '一个人在北京漂着，习惯了。你们呢？', 0, 0, 55, 2, 1, '2026-08-30 10:37:57.046', '2026-08-30 10:07:01.813', '2026-08-30 10:37:57.046', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10006, 10, 10002, 'id10272+霓裳羽衣+申请皇家贵族', '家族名：霓裳羽衣\n家族宗旨：以舞会友，以歌传情\n申请类型：皇家贵族\n家族成员：28人，日活跃15+，望管理员批准！', 0, 0, 93, 2, 1, '2026-08-30 10:37:17.439', '2026-08-30 10:07:01.831', '2026-08-30 10:37:17.439', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10007, 17, 10003, '建议增加勋章系统', '建议社区增加勋章系统：签到达人、灌水狂人、人气之星……帖子被加精华也有勋章，这样大家更有动力。', 0, 0, 43, 1, 1, '2026-08-28 10:07:01.677', '2026-08-30 10:07:01.850', '2026-08-30 10:07:01.859', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10008, 9, 10005, '鸿蒙3.0发布日期确认，首批华为、荣耀适配机型已公布', '如题，官方公布了首批适配名单，快看看有没有你的机型。', 0, 0, 81, 1, 1, '2026-08-29 01:07:01.677', '2026-08-30 10:07:01.869', '2026-08-30 10:07:01.880', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10009, 26, 10000, '【精武堂】第二届武林大会报名帖', '第二届武林大会即日起开放报名！\n赛制：32进16单败淘汰，每天3场，周日决赛。\n奖励：冠军专属马甲+500金币，亚军300金币。\n回帖格式：【报名】游戏ID+常用武器。', 0, 1, 81, 3, 1, '2026-08-30 14:44:40.034', '2026-08-30 12:33:44.042', '2026-08-30 14:44:40.034', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10010, 22, 10000, '晒花大赛第3期：谁的花最惊艳', '本周主题：玫瑰！\n把你的花园截图发上来，点赞最高的送高级花种×10。', 0, 1, 75, 1, 1, '2026-08-30 11:33:44.071', '2026-08-30 12:33:44.066', '2026-08-30 12:33:44.076', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10011, 25, 10000, '车神争霸赛：谁的车最贵', '晒出你的座驾！劳斯莱斯幻影镇楼，不服来战。', 0, 1, 67, 1, 1, '2026-08-30 11:33:44.090', '2026-08-30 12:33:44.086', '2026-08-30 12:33:44.096', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10012, 20, 10000, '【新区】虎年新区开服公告', '虎年新区正式开服！\n开服前3天经验翻倍，冲级榜前10名送神兵利器。', 0, 1, 70, 1, 1, '2026-08-30 11:33:44.112', '2026-08-30 12:33:44.107', '2026-08-30 12:33:44.117', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10013, 32, 35797804, '【公坛区域】管理须知', '【公坛区域管理须知】\n\n一、公坛区域范围\n公共论坛下设各板块（休闲灌水、时尚美眉、新人求助、情感天地、数码动漫、公坛事务）均属公坛区域管理范围。\n\n二、管理员职责\n1. 公坛协管员负责日常巡查，及时处理违规帖；\n2. 各板块版主负责本版加精、置顶、删帖；\n3. 发现宣传外网、广告、辱骂等行为，第一时间删帖并上报客服中心。\n\n三、发帖规范\n1. 禁止发布外网链接与广告；\n2. 禁止人身攻击、地域攻击；\n3. 水贴适度，共同维护社区环境。\n\n四、本须知自发布之日起施行，解释权归公坛管理组。\n\n　　　　　　　　家园社区 公坛管理组', 1, 1, 527, 1, 1, '2026-08-30 13:44:10.081', '2026-08-30 13:44:10.070', '2026-08-30 14:32:53.556', 0, 0, 1, 0, 0);
INSERT INTO `threads` VALUES (10014, 32, 35797804, '【公坛区域管理】在职管理员花名册', '【公坛区域管理】在职管理员花名册\n\n公坛组长：站长小Q（10000）\n公坛协管员：　　瞿詺南　（35797804）\n\n各板块版主：\n休闲灌水：云起（10001）\n时尚美眉：安珞（10002）\n\n（本花名册由公坛管理组维护，任免实时更新，如有变动请回帖公示）', 1, 1, 317, 0, 1, NULL, '2026-08-30 13:44:10.086', '2026-08-30 14:32:55.327', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10015, 7, 35805083, '【小破网】社区论坛公约', '【小破网】社区论坛公约\n\n第一条 为加强社区网络平台的建设、管理及维护，营造健康和谐的交流环境，特制定本公约。\n第二条 友友发帖回帖应当遵守法律法规，尊重他人，文明用语。\n第三条 禁止发布外网链接、广告及任何形式的商业推广。\n第四条 各板块版主应当及时处理违规内容，公坛协管员负责巡查督导。\n第五条 本公约自发布之日起施行。\n\n　　　　　　　　家园社区 公坛管理组', 0, 1, 79, 0, 1, NULL, '2026-06-01 09:00:00.000', '2026-08-30 14:38:47.172', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10016, 7, 35805083, '严厉打击宣传外网公告', '【公告】严厉打击宣传外网行为\n\n近期发现有部分账号在社区内宣传外部网站、发布拉人广告，严重扰乱社区秩序。\n\n现公告如下：\n一、凡发布外网宣传内容的帖子一律删除，账号视情节封禁3-30天；\n二、屡教不改者永久封号并公示；\n三、欢迎大家向客服中心举报，举报属实奖励金币。\n\n家园是我们共同的家，请大家一起守护！\n\n　　　　　　　　家园社区 客服团', 0, 1, 527, 2, 1, '2026-07-02 09:00:00.000', '2026-07-01 08:30:00.000', '2026-08-30 14:14:15.408', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10017, 7, 35806088, '【UBB代码演示】ubb功能调用代码', '【UBB代码演示】\n\nUBB 是家园发帖常用的排版代码，掌握以下几条，帖子立刻好看十倍：\n\n[b]加粗文字[/b] —— 加粗\n[i]倾斜文字[/i] —— 倾斜\n[color=red]红字[/color] —— 变色\n[url=http://家园社区]链接[/url] —— 超链接\n\n排版三件套：分割线、颜色代码、居中标签，配合使用效果最佳。\n大家回帖练手，不懂就问！', 0, 1, 8467, 3, 1, '2026-05-22 12:00:00.000', '2026-05-20 14:00:00.000', '2026-08-30 14:14:15.438', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10018, 7, 35806260, '密码忘记了', '呜呜呜，密码忘记了怎么办？号码还在的，求管理员帮忙找回！', 0, 0, 67, 1, 1, '2026-08-21 09:00:00.000', '2026-08-20 21:00:00.000', '2026-08-30 14:14:15.460', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10019, 5, 35806114, '【休闲灌水】论坛版规', '【休闲灌水】论坛版规\n\n一、本版为休闲灌水专区，日常聊天、灌水打卡均可；\n二、禁止发布广告、外链及人身攻击内容；\n三、恶意刷屏者版主有权删帖并禁言；\n四、精华帖标准：有内容、有温度、有回复量；\n五、本版规最终解释权归公坛管理组。\n\n祝大家灌水愉快！', 1, 1, 257, 1, 1, '2026-06-16 08:30:00.000', '2026-06-15 10:00:00.000', '2026-08-30 14:14:15.481', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10020, 5, 10001, '春趣', '春风拂柳绿如烟，\n燕归来时花满天。\n童子放鸢田埂上，\n一竿撑起半边天。', 0, 0, 6, 2, 1, '2026-08-29 12:00:00.000', '2026-08-29 11:00:00.000', '2026-08-30 14:14:15.506', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10021, 5, 10001, '咏荷', '接天莲叶无穷碧，\n映日荷花别样红。\n咏而归舟惊鹭起，\n一池清香伴晚风。', 0, 0, 2, 0, 1, NULL, '2026-08-29 11:20:00.000', '2026-08-30 14:14:15.523', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10022, 5, 10001, '怎知', '怎知相思苦，\n只缘未相逢。\n若得君心意，\n不负此生情。', 0, 0, 3, 0, 1, NULL, '2026-08-29 11:40:00.000', '2026-08-30 14:14:15.538', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10023, 5, 10001, '西风', '西风昨夜过园林，\n吹落黄花满地金。\n游子他乡逢秋雨，\n一封家书抵万金。', 0, 0, 5, 0, 1, NULL, '2026-08-29 12:00:00.000', '2026-08-30 14:14:15.553', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10024, 5, 10001, '桃花', '桃花坞里桃花庵，\n桃花庵下桃花仙。\n桃花仙人种桃树，\n又摘桃花换酒钱。', 0, 0, 3, 0, 1, NULL, '2026-08-29 12:20:00.000', '2026-08-30 14:14:15.567', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10025, 8, 35806066, '终是庄周梦了蝶，你是恩赐也是劫。', '庄生晓梦迷蝴蝶。\n有人说，庄周梦的是蝶，而我梦的是你。\n\n你是恩赐，也是劫。\n恩赐是遇见，劫是离别。\n\n来情感天地，说说你的故事。愿天下有情人都不留遗憾。', 0, 0, 89, 2, 1, '2026-08-26 20:00:00.000', '2026-08-25 22:00:00.000', '2026-08-30 14:14:15.592', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10026, 8, 10003, '执子之手 与子成双', '执子之手，与子成双。\n征婚台首帖，希望在这里遇到那个愿意陪我赏花看月亮的人。\n要求不高：上线勤、说话暖、会陪聊。', 0, 0, 66, 1, 1, '2026-08-23 10:00:00.000', '2026-08-22 19:00:00.000', '2026-08-30 14:14:15.611', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10027, 8, 35806205, '有没有', '有没有那么一个人，你一上线就想看他在不在线。\n有没有？', 0, 0, 30, 0, 1, NULL, '2026-08-27 23:00:00.000', '2026-08-30 14:14:15.625', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10028, 11, 35805768, '4月份封控19天总结', '【与世无争】家族4月份封控19天总结\n\n19天里，家族成员线上陪伴不断：\n- 每晚八点聊天室准时集合，累计发言3000+条；\n- 家族互赠金币486次；\n- 新增成员7人。\n\n隔离病毒不隔离爱，与世无争，与家有约。', 0, 0, 88, 2, 1, '2026-08-06 12:00:00.000', '2026-08-05 20:00:00.000', '2026-08-30 14:14:15.650', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10029, 11, 35806199, '恭喜我大咪', '恭喜家族的大咪当选本周风云人物，撒花！\n大咪上周发帖30篇、回帖200+，勤劳榜第一，实至名归！', 0, 0, 40, 2, 1, '2026-08-19 10:00:00.000', '2026-08-18 21:00:00.000', '2026-08-30 14:14:15.675', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10030, 10, 35806066, '我要申家', 'ID:35806066+断念+申请创建家族【断念阁】\n\n家族宗旨：聚是一团火，散是满天星。\n现有成员：15人，日活跃8+。\n望管理员批准！', 0, 0, 70, 1, 1, '2026-08-09 09:00:00.000', '2026-08-08 15:00:00.000', '2026-08-30 14:14:15.695', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10031, 9, 35806077, '10000毫安电量+加密芯片+3D人脸+128GB', '如题，这配置放在当年想都不敢想。\n现在新机内卷到这个程度了，大家换机是选大电池还是选影像？', 0, 0, 60, 1, 1, '2026-08-26 14:00:00.000', '2026-08-26 12:00:00.000', '2026-08-30 14:14:15.715', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10032, 9, 35806077, '荣耀再次爆发，6000mAh“神机”备货，天玑9000加持', '如题，官方公布了首批适配名单，快看看有没有你的机型。', 0, 0, 54, 1, 1, '2026-08-24 11:00:00.000', '2026-08-24 10:00:00.000', '2026-08-30 14:14:15.737', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10033, 32, 35797804, '【小破网】社区统筹管理纲要', '第一章，总则\n\n第一条 为加强社区网络平台的建设、管理及维护（以下简称社区），服务社区工作，服务用户，提升社区知名度、美誉度，特制定本管理规定；\n\n第二条 网络平台建设、运营和管理必须坚持立德树人，以人为本，弘扬社会主义核心价值观，传递网络正能量；\n\n第三条 社区目前所建设和管理的网络平台包括：\n\n1.社区网站，网址：https://家园社区\n\n2.手机QQ群，群号242973387\n\n第四条 社区发布的信息内容须遵循真实性、准确性和及时性相统一的原则；\n\n第五条 信息内容和展现方式须全面、完整、合时宜。\n\n第二章 管理机构及职责\n\n第六条 社区设立  七大管理区域，即〔公坛〕〔同城〕〔客服团〕〔家服团〕〔爱游团〕〔社区传媒〕〔监察局〕\n\n第七条，各区域管理 对所负责的区域进行全面监督及管理。\n\n第八条 专职监察员负责社区的信息收集与汇总、日常管理，负责社区案件审核，处理通告发布以及其他信息材料的管理、录入与发布；\n\n第九条 处理结果发布后的反馈（含反响评论、错误指正、批评建议等）须及时上报，妥善处理。\n\n第三章 规范及要求\n\n第十条 信息发布严格执行\"谁发布、谁负责；谁批准、谁负责\"的原则。如发布不良、有害或反动等内容信息，将对经办人和负责人依据规定追究相关责任；\n\n第十六条 其他事宜参考【小破网】社区论坛公约\n\n第四章，社区的应聘要求及流程\n\n随着社区人员越来越多，管理人员的更替和加入也越来越频繁。社区必须做出一个统一的管理流程，终结各自一摊的局面。\n\n为此做出以下规定，\n\n应聘贴必须在招聘活动进行时有效，拒绝凭空应聘和空降职务。\n\n应聘要求统一为〔一线管理〕\n\n1.家园等级不低于3级\n\n2.论坛等级不低于一级\n\n3.有足够时间上线活跃\n\n〔一级管理〕\n\n不接受直接申请，\n\n应聘帖子统一为\n\n标题：昵称+ID+应聘区域职务\n\n帖子内容为\n\n①.家园ID:\n\n②.家园昵称:\n\n③.家园等级:\n\n④.论坛等级:\n\n⑤.是否已经绑定手机:\n\n⑥.每日在线时长:\n\n⑦.对社区的了解:\n\n⑧.对应聘职位的了解:\n\n社区原则上是不支持社区管理兼职，但是各区域可根据实际情况调整管理是否兼职，最多两个一线管理职务，需要分管仲裁报备到仲裁团，(家族职务除外)\n\n第五章， 管理员的考核和审核\n\n报名成功后：需要统一安排培训，是否被录取，以录取公告为准。\n\n统一为不带权考核时间最多为7天，\n\n带权实习最长时间为一个月，\n\n第六章，管理员实习\n\n社区支持新人培训后上岗，但是各区域可根据实际情况选择是否培训。需要分管仲裁报备到仲裁团。\n\n实习内容包括\n\n1.实习目的\n\n2.实习时间\n\n3.实习单位\n\n4.实习主要内容\n\n5.实习心得\n\n撰写报告后提交各区域内版\n\n管理的晋升参考社区管理统筹大纲\n\n【监察局】\n\n第七章，监察员转正后的工作内容详责\n\n第一条 ，工作内容可参考【小破网】社区论坛公约进行，\n\n二条， 关于签名\n\n1、 不得出现宣扬反动、封建迷信、淫秽、色情、暴力、凶杀、恐怖、教唆犯罪等不符合国家法律规定的以及任何包含种族、性别、宗教歧视性和猥亵性的信息内容；\n\n2、 不得出现有侮辱性言语、挑衅、辱骂其他人以及不健康内容；\n\n3、 不得出现国家明令禁止广告的内容或链接；\n\n4、 不得出现其它违反《小破网社区各区域管理规定》的内容。\n\n　　　　　　　　家园社区 公坛管理组', 0, 1, 311, 2, 1, '2026-07-16 09:00:00.000', '2026-07-15 16:00:00.000', '2026-08-30 14:14:15.764', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10034, 32, 10000, '[重要]家园社区已完成所有法定备案', '家园社区（家园社区）已取得工信部备案：渝ICP备17001534号-2。\n\n备案信息可在工信部官网查询。家园合法合规运营，请大家放心游玩，也别忘了身边的老朋友。\n\n　　　　　　　　家园社区 站长办', 0, 1, 4386, 2, 1, '2026-06-10 14:00:00.000', '2026-06-10 10:00:00.000', '2026-08-30 14:14:15.794', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10035, 5, 10000, '表情测试帖', '大家好呀！/微笑 今天心情不错/呲牙 大家一起来灌水/鼓掌', 1, 1, 8, 1, 1, '2026-09-04 13:35:26.881', '2026-08-30 19:19:34.830', '2026-09-04 13:35:26.881', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10036, 33, 10001, '【清风明月·中秋活动】月圆人团圆，回帖赢金币', '中秋佳节，家族全体成员一起赏月吃月饼，回帖即可获得金币奖励！', 0, 0, 287, 0, 1, NULL, '2026-09-05 10:15:54.893', '2026-09-05 10:15:54.893', 0, 0, 0, 0, 0);
INSERT INTO `threads` VALUES (10037, 33, 10002, '【与世无争·周末聚会】来家族大厅唠唠嗑', '周末啦，兄弟姐妹们快来家族大看台集合，聊聊这一周的趣事～', 0, 0, 201, 0, 1, NULL, '2026-09-05 10:15:54.898', '2026-09-05 10:15:54.898', 1, 0, 0, 100, 0);
INSERT INTO `threads` VALUES (10038, 33, 35797804, '【断念阁】新成员入阁欢迎帖', '欢迎新伙伴加入断念阁，新人报道帖～', 0, 0, 142, 0, 1, NULL, '2026-09-05 10:15:54.902', '2026-09-05 10:15:54.902', 2, 0, 0, 100, 2);

-- ----------------------------
-- Table structure for ttou_applies
-- ----------------------------
DROP TABLE IF EXISTS `ttou_applies`;
CREATE TABLE `ttou_applies`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `slogan` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `status` bigint NULL DEFAULT 0,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_ttou_applies_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of ttou_applies
-- ----------------------------
INSERT INTO `ttou_applies` VALUES (1, 10000, '人生海海，山山而川，不过尔尔。', 0, '2026-09-05 20:21:55.416');

-- ----------------------------
-- Table structure for ttou_worships
-- ----------------------------
DROP TABLE IF EXISTS `ttou_worships`;
CREATE TABLE `ttou_worships`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `target_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_ttou_worships_target_id`(`target_id` ASC) USING BTREE,
  INDEX `idx_ttou_worships_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of ttou_worships
-- ----------------------------

-- ----------------------------
-- Table structure for user_badges
-- ----------------------------
DROP TABLE IF EXISTS `user_badges`;
CREATE TABLE `user_badges`  (
  `user_id` bigint UNSIGNED NOT NULL,
  `badge_id` bigint UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `badge_id`) USING BTREE,
  INDEX `fk_user_badges_badge`(`badge_id` ASC) USING BTREE,
  CONSTRAINT `fk_user_badges_badge` FOREIGN KEY (`badge_id`) REFERENCES `badges` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `fk_user_badges_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of user_badges
-- ----------------------------
INSERT INTO `user_badges` VALUES (10000, 12);
INSERT INTO `user_badges` VALUES (10001, 12);
INSERT INTO `user_badges` VALUES (35797804, 12);
INSERT INTO `user_badges` VALUES (10000, 13);
INSERT INTO `user_badges` VALUES (35806114, 13);
INSERT INTO `user_badges` VALUES (35805083, 14);
INSERT INTO `user_badges` VALUES (10002, 15);
INSERT INTO `user_badges` VALUES (35805083, 15);
INSERT INTO `user_badges` VALUES (10001, 16);
INSERT INTO `user_badges` VALUES (10004, 16);
INSERT INTO `user_badges` VALUES (35806199, 16);
INSERT INTO `user_badges` VALUES (10003, 17);
INSERT INTO `user_badges` VALUES (35797804, 17);
INSERT INTO `user_badges` VALUES (35806088, 17);
INSERT INTO `user_badges` VALUES (10000, 18);
INSERT INTO `user_badges` VALUES (10001, 18);
INSERT INTO `user_badges` VALUES (10002, 19);
INSERT INTO `user_badges` VALUES (10005, 19);
INSERT INTO `user_badges` VALUES (10005, 20);
INSERT INTO `user_badges` VALUES (35805768, 20);

-- ----------------------------
-- Table structure for user_flowers
-- ----------------------------
DROP TABLE IF EXISTS `user_flowers`;
CREATE TABLE `user_flowers`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `flower` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `count` bigint NULL DEFAULT 0,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_uf`(`user_id` ASC, `flower` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 13 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of user_flowers
-- ----------------------------
INSERT INTO `user_flowers` VALUES (1, 10007, '', 1);
INSERT INTO `user_flowers` VALUES (2, 10001, '', 1);
INSERT INTO `user_flowers` VALUES (3, 10000, '太阳神花', 1);
INSERT INTO `user_flowers` VALUES (5, 10000, '月光花', 0);
INSERT INTO `user_flowers` VALUES (6, 10007, '太阳神花', 8);
INSERT INTO `user_flowers` VALUES (7, 10007, '金色向日葵', 7);
INSERT INTO `user_flowers` VALUES (8, 10007, '蓝色妖姬', 12);
INSERT INTO `user_flowers` VALUES (9, 10007, '粉郁金香', 1);
INSERT INTO `user_flowers` VALUES (10, 10007, '黑郁金香', 3);
INSERT INTO `user_flowers` VALUES (11, 10007, '七彩向日葵', 1);
INSERT INTO `user_flowers` VALUES (12, 10007, '???', 5);

-- ----------------------------
-- Table structure for user_goods
-- ----------------------------
DROP TABLE IF EXISTS `user_goods`;
CREATE TABLE `user_goods`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `good_id` bigint UNSIGNED NULL DEFAULT NULL,
  `count` bigint NULL DEFAULT 0,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_ug`(`user_id` ASC, `good_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of user_goods
-- ----------------------------
INSERT INTO `user_goods` VALUES (1, 10000, 1, 1, '2026-09-05 16:08:19.762');
INSERT INTO `user_goods` VALUES (2, 10000, 7, 4, '2026-09-05 16:37:56.239');

-- ----------------------------
-- Table structure for user_roles
-- ----------------------------
DROP TABLE IF EXISTS `user_roles`;
CREATE TABLE `user_roles`  (
  `user_id` bigint UNSIGNED NOT NULL,
  `role_id` bigint UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `role_id`) USING BTREE,
  INDEX `fk_user_roles_role`(`role_id` ASC) USING BTREE,
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of user_roles
-- ----------------------------
INSERT INTO `user_roles` VALUES (10000, 1);
INSERT INTO `user_roles` VALUES (35797804, 3);
INSERT INTO `user_roles` VALUES (10001, 4);
INSERT INTO `user_roles` VALUES (10002, 4);
INSERT INTO `user_roles` VALUES (10003, 4);
INSERT INTO `user_roles` VALUES (10004, 4);
INSERT INTO `user_roles` VALUES (10005, 4);
INSERT INTO `user_roles` VALUES (10006, 4);
INSERT INTO `user_roles` VALUES (10007, 4);
INSERT INTO `user_roles` VALUES (35792031, 4);
INSERT INTO `user_roles` VALUES (35799015, 4);
INSERT INTO `user_roles` VALUES (35803370, 4);
INSERT INTO `user_roles` VALUES (35804196, 4);
INSERT INTO `user_roles` VALUES (35805083, 4);
INSERT INTO `user_roles` VALUES (35805532, 4);
INSERT INTO `user_roles` VALUES (35805768, 4);
INSERT INTO `user_roles` VALUES (35806066, 4);
INSERT INTO `user_roles` VALUES (35806077, 4);
INSERT INTO `user_roles` VALUES (35806088, 4);
INSERT INTO `user_roles` VALUES (35806114, 4);
INSERT INTO `user_roles` VALUES (35806199, 4);
INSERT INTO `user_roles` VALUES (35806205, 4);
INSERT INTO `user_roles` VALUES (35806260, 4);
INSERT INTO `user_roles` VALUES (35806345, 4);

-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `nickname` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `password` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `gender` bigint NULL DEFAULT 1,
  `signature` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `color` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `coins` bigint NULL DEFAULT 0,
  `exp` bigint NULL DEFAULT 0,
  `level` bigint NULL DEFAULT 1,
  `status` bigint NULL DEFAULT 1,
  `last_active_at` datetime(3) NULL DEFAULT NULL,
  `last_login_at` datetime(3) NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `avatar` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `noble` bigint NULL DEFAULT 0,
  `partner_id` bigint UNSIGNED NULL DEFAULT 0,
  `baby_name` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `achieve` bigint NULL DEFAULT 0,
  `priv_id` bigint UNSIGNED NULL DEFAULT 0,
  `garden_pots` bigint NULL DEFAULT 4,
  `noble_exp` bigint NULL DEFAULT 0,
  `blue_lv` bigint NULL DEFAULT 0,
  `blue_exp` bigint NULL DEFAULT 0,
  `qq_lv` bigint NULL DEFAULT 0,
  `qq_exp` bigint NULL DEFAULT 0,
  `blue_start` datetime(3) NULL DEFAULT NULL,
  `blue_end` datetime(3) NULL DEFAULT NULL,
  `qq_start` datetime(3) NULL DEFAULT NULL,
  `qq_end` datetime(3) NULL DEFAULT NULL,
  `city` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `active_days` double NULL DEFAULT 0,
  `last_active_date` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `yuanbao` bigint NULL DEFAULT 0,
  `jinzuan` bigint NULL DEFAULT 0,
  `youquan` bigint NULL DEFAULT 0,
  `age` bigint NULL DEFAULT 0,
  `birth_year` bigint NULL DEFAULT 0,
  `birth_month` bigint NULL DEFAULT 0,
  `birth_day` bigint NULL DEFAULT 0,
  `introduction` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `avatar_base64` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `idx_users_username`(`username` ASC) USING BTREE,
  UNIQUE INDEX `idx_users_nickname`(`nickname` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 35806346 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of users
-- ----------------------------
INSERT INTO `users` VALUES (10000, '10000', '站长小Q', '$2a$10$sD9fZ//aaD3s5LheftYGQOogtRRfjQTXIhQjL6C3pmIc1cUuN6.QW', 1, '', '', 1363, 9060, 13, 1, '2026-09-07 16:04:38.583', '2026-09-07 16:04:38.583', '2026-01-01 08:00:00.000', '2026-09-07 15:39:09.524', '001130576.jpeg', 1, 0, NULL, 906, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '', 7.3999999999999995, '2026-09-07', 8, 2, 20, 0, 0, 0, 0, '', '');
INSERT INTO `users` VALUES (10001, '10001', '云起', '$2a$10$3etMd.obpzuGchSXlC6DZ.ToNcyE5AAX6Yx.vhWE2.ut8j0dKDZdC', 1, '云起时，风也温柔', '', 159, 3255, 8, 1, '2026-09-06 22:17:03.000', '2026-09-06 22:17:03.000', '2026-01-15 09:00:00.000', '2026-01-15 09:00:00.000', '', 2, 10002, '小云朵', 502, 322, 5, 100, 1, 200, 1, 300, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-11-03 20:08:36.000', '', 16.8, '2026-09-05', 0, 0, 0, 28, 1998, 9, 5, '说明', 'data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT/2wBDAQMEBAUEBQkFBQkUDQsNFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCABkAGQDASIAAhEBAxEB/8QAHgAAAQQDAQEBAAAAAAAAAAAABQAGBwgBAwQJAgr/xABBEAABAwIEAwYEAwUECwAAAAABAgMEBREABhIhBzFBCBMUIlFhMnGBkRWhsSNCYtHwCSRSchYlM0Njc4Ki0+Hx/8QAGgEAAwADAQAAAAAAAAAAAAAAAgMEAAEFBv/EACYRAAEEAgICAgIDAQAAAAAAAAEAAgMREiEEMRNRIkEycRRhgZH/6wJSAAAAAfrpOxJtwztfpU/hzUchNjUAAAI4bWU6p+WWTHrHsd1hdugt30KGnU7XDPsKEnJmtwkBQnrLRjS8XQiNcHn0TMVuVNpUwPM95/t+V0wBMiejT7jpWGxM17X3eEGRVTB3Hvst0Agds6LGKv9tLAPhmcJZMaIyVed9f00YWTp7obhdAF6mzGA/AomHSFyvzS5zoe8+4hfIUJWjIu9XY93/2u1Qh6gKhjzWbl9BBwBRG6hYoDE0E0lNH5FM79dt7ktr/LstCam4+c3yn9Z8/pUf/cgV8Px3Z2v8kBa3hjp2BIxRDuYoO4VYrn3KjYYIM4RZW4879vZHvNOil0SuhgHe2yeL2lsHsusvMjIhgNCZNCaJdi2fBbm0VhbhRjT6xS1WIqX2zA0bbYeKCmauiQI3YDGEtSR2aDXA3wZmxwCBccVPc+ds6fH10GTcglnrKCeEaZdeNjQkiCE7k5ynCdHaa5A0VmTpcZ27KNCjGN+gbhDKdnRJeowuLOUkiDkgwMnhfHxxA52EmYQskBvT2upc9qW7dTpt3udyCjKVopkoZNnzOxs5FJO1HgMa4s7xLAa74MNynAvX1qpH+PF0m1i2FPTcUd/v4xncKDvW1DD1Ixl51L7npSXawbayNFEkAo3r1PnBBQ0IQ18WVC63u8pWtoimhVJZr6qSg/Ifjmuzol2rwKP2scsBMxXN3ojSetlXNCpvm5otb/wGaun9ZklmSxwDdnM7qYRRxvfrU8WaW7Ff20tebRRQe36IiNAaV70Kcui7cc6MmMN1ISLOxv/aAAwDAQACEQMRAD8ALyF88MuncPKXSczOVyMt9uUtbjikd55CV3vt9cOx53bHEuRZRGPN7F0rhtaMx1o0GhTqho7zwzRdKbXuBz/LDVyhxZj5qrbNOQ0ptTjSnApSCnl05nDvMoDqCcai+hKgQkBXrbAYpwdqqTVzPkCZXc4xauzNaZZZLRU2q9zpVc2xIBkBPXAkSbqG9sZVL0nc42SSACtoqqSoHnYYwmXpJscCFTSobmw98aRNA5HfGALE40S7AX54JwZYunDNFS0gb39sEINS86TffDgCsUjQpQSBvvhoZ6mPy68zG8U6iOGwShKrJ5E8sdkKp3tdW+GrmmoB3M3xbJb/AD0n+eDultuytkGvWjJDcdlKf47qP3wsBIVVWxHQhLlgB0wsU0EvIodQ+JVIzYHjT31L7lQQsOIKDci45465FZYZWnvXm29RsnWoC59sVuyNV1ZczHU4TjidaghQIvYlPz9j+WNfFWufi4huBd1NqUB7XH/rCfHZSXHFWTNQBFwse1jfGs1H+LFOYtfnwk2jzX2QOiHCP0x2pzpXNrVaZt/x1fzwXhKESq3Caj5j598fLlXZasXHUoubAqNrnFUms9V0EE1WST7uHGZuZ6jVSnxkt2QhKgoJUo2BHW3LqfvgfCjEqtjmZx3KmV2cwVVKodMkK0MOuggvm1/IOu3XlgDlPP2Xc3zTEbq3gZa/9mJMdZQo+mpGq32xAOeeLFdz8zS4NTmOLpNKaEeDBSEhthva9gALk6Rud9sA8vvSGpraoTwTKBuhKtrn2OMEJx2dpglblVWFcnM2Vqzk6Qy1VYamUvo7xh4HU28j/EhQ2P6jrbA2PUC04B+8D1w6uHXEabxd4JS8k5qeZiVBsqdptTkktrZdR0vyO9gQdJsTz2u08o8JcwP5enzWJbc8QlJDzylKOrULpIJuSNJFjyO1jiQS4al0VV4XO3GLCNxaxZQF7HDazBUbVp9er/dE/wDbgCc3RqTLcjTZLTL7StK0FwGx+nPAKsZvhypr62nwsKbUkEYqq+lO11E2nJHqH7Ib3wsMZivkNjYfc4WHUgUDt1OTIfckOvKW+bDWedrcsa5k91wIC1qUm/InE81bsB8fcuQnZcrhzNfZQL/6vlRpayPZDLilH6DEL5kyJmLLckRq1QqnR3wCrup8NxlVhsdlJBxUAAoyg4eScfaXL8ji5HY47PmR+0hwzrGWs0U16i1WlTC+zmyEA08hpxF0tr1DS6nUheyhcXFlAbYOVH+zcy4/MmRaHxLnS3GXFAOP0ZKW1JtsAQ9c7g3Nh8uuMYfIcQie3xgOcdEWqQoeA9sHMs005grUKnINlSXUtX+ZxJ3aR4IUbgS1luhsSJNRq0lDsmZUXBoSsAhKEoRchIHm9T78rRJRKgKRVIk5krDkd1Lo81twb9LH7HAvaRY+1ppFgnpbanRpNFnyIUxtUeUyspWhYtyxpQUtqBNwR1GLR8QaC1mumU1NRpkaZVUoGtyKsNh1N/iCykkXG+nl74OZD4IZIqNLdL9B0yNBBU6+p3QbcxyH5YljnDqvtdGTiOaTidIsxnFimdkBS47lOdqjbbCUSGdUl+Q4oO962pZsWlJQoK2JFmdNgpQOK8cLuLlWoFYabcSic3KQ3EU68XC4ykL8pQUKBuALWN0kbFJwdzLRv9BIC6OhsiW+28wlQV+zSVlSVKUN99IFrfXlix3ZHoeQOF0KlVLMFPRUMzGQZENaEhTqVFAQne2yd1bX98KlLGNNtu0bGvc5tO6VdOKlEkszJk5VKeab79V3lIAHPkkp2Ivc7i/5nEbtzbnnYdL49qalk7hlxVy3LgzaBTB41J71+G0208lR/fSsb3vv1B6gjHj9x04dyeEPFjMWU5Cg54CRZl5IsHmVALact01IUk26EkdMb47gW43dJHJFvLgCLQhmqaWwLA/XCw3BJ98LFVJFhfoFy1mpqr0h58qSlUZNnPQG3PDdi8TIz0NxMosynApaVJDZCSn0BIsdv5YhqNW15fivMNzXHErJ8QpC9lkc0m30x0xoiW2kPIbNt02HT3tisMJ1fSQSB9I/mOvyM0sKiMNpgxBrLcdpISkWFhcD9PYYE0uA0ju1Ib0ktJUCRuD1B+/644PxxtsjS8FuJVfy323tY4JO1lptWtSkpUUKuBy+eL4semhc+YO7Krn2oOyo9x5rVCqzOYYOXY8Dvo8l6a2t1akEpUO7QgXURZV7kcx74jmi8G+GfBCC3MjAZlzK2paTOrTSFMJ28q2GgVNg9fPqUCBYjli45zNBzDlZVLqVFlSEtoLzUqAFNPJbUSQvVYhQ3vci17YpbxLomXqPmR6oRXqhWGUuXZh1WS2VOrA31pbSkEA8wbjle97Y8tyJ3TTODXfFep43GZHE0vb8v76QPOOem6/mQSoRE2UpIeUgOhKl7+b578/vgkzxOOV6SqRIiaVlJUWmjrJPtYb4jubPifjDkdiGwmW+hK5DEFCG2GgLgbdTudzf54I1KgusUQpdioQoo1JkMgWFuVwCenUbewOCjis6T5JXBlFCalXV1uY7U6g2A+4o92yrpcEDrz3+mHBkOuvUh+S6t3v1d33LLi/iTc2UR6bXF+YviHJNWRGUp7vlvOc0CxO/TGMv5+ciSkNSUPNbfGUHTe9/6+eKsHFcvytva9COAFZ8TU0u1CXLUi4Q2hl0pIHIBKdKgB9vqThmf2ofD2l/hGSs807S1KAVSZdyCt5FlONKJA3KbOAk/wCJI6Ya3BHPUdVKRP75KpJu2tZSpXdgfw8iTz3xKXGXI03tG8LEZcp82KipsyEvxpVQKmkJ0khSVWCiDpum9tyB64KJpdY6pDKRQPteazMljuxqUkn3BwsTi52MMwNLUheZcv6kkglLz1j8rtjCxu2+0qz6Vi84Z7jZLWZFOfenwnljvGlaWlLeJ8ywm5J2Avcj1w8Mv8fY7HhELc7gPAJAdO+s/wBDfDDrPAiVLyWl2o1Onwq9pD6IDjhCGW7gKWtxXUJIUeXMfPAPNnCLLVEy1RqghVczU0thbKa3SZIVGakAm/8Ad0I16ARsrWb39dsdBkTsrBpVSwShtuZoe1ZimJh1FvvI8vw6VJuGUgFCjzvfHzS8+0jIuYpkmeE1F+AnUEMuG2uwPlCk+awV1Itzt6RNkekVjK9Bpr8rNDEuApCFpE6Gph9KSlNyo6uiioAad/KBckgCM8z6TmqVVXGYUiPWu4XCVLb3DhSRsRewBtYnnYkXxZ4wWgDS5rmO4z8pm2Pr9qR+Ovaga4lUNNHptLn06MFhxTxfShZUNx8BJtfe1x05YqPVcl0+ZLXIaSth9ZN1NlaAT+mJO4fcFKlmGUkT6jEoUGxKpT6lrT/lSnUBc9L2Hv0x15noOTsruqbbqsyqONm1koSjWfZIJt9VY03jxnRCidyn9g0FWuvcLJjqpDtPnPBx22oBe5tuOuJ04ENsOQ4NJqcOfJnoIEhwMqC3D6IJSRytYe3LDsyzlvMdecaey/w9vTlGxrGYJPhobQ9dShZVuekFRPocTlRuKVG4bxHWMzcSlTp7LSQINJilinp8tyEjWFG2wuCj/Kd8SzQNuou0+KdxFyHSYFe4YUzPuVvw6s02NSzTHC4lt2N3K1tEqKtK7BWvqL3vuOdgfvOfCbgNUsqZbtQ0U6qSIZguVCmyVNhDiEpSHlI1aSu9rlSfMVKvuL4bHEztlZUrNWYjDNmYY7ratPiYTUezab+pRqI9ivEVZs4j5nYiGTTM4xc30B50hBk2MhvoO9ac1EXtzSpSffEH8ea6yVo5ER2WKXsrZFyjkasVGjUSTU1JWpYRJnrbWVja1ilAsLXN+VjyxIGWJaExXI8ixZeSW0vtDu177EKA5KBNjyI58t8VQg53qanWJElQStrdLkVQNrD0sRt6WGJTyjxRiyHVFwhRdHnDYuFG3PTzvba/pa5xVDG6N1naU+RkgoaXIluTPLinltuvNOLZcu8pOhSVFJT9x+eFhm1SpS3Mx12RFdWiNInuvNhAJFidz9Tc/XCxWIowKwUJkdf5KVc75unUeNIEx5s1V9ADjTaf2aVpSAm421WsOZ+2IqyDJz/RGX4tLnOzkOvqldymNr7tZPmUkj4Qeo5YtBw47GtYzvWXq5nNb9OiFRW3C1WcePTXvsOWLS0DgXlDL1NQxT6SiE6EC0hlRS6lQHxBYsb++Hy8iOwyMaCON3Jc8zSP+RXmzWX8x+IZerlWlSavIWGY9PjO6ApZNhcJCQT8/vi2PDvsh1KDw8Ydr0wKzK6tLvcoWVNspJvpUobqO+9j8r45e0r2eoWUMmDNtEZ8fXI09l2TNkkBwIUsjypHl+JSL7XsCb4sDwZzUut5JpjNQU4mpsspQ8JCrrWbfEb73IxI/lfOm6RmN0guU5KKU9lCbVaciPVsxiBH1gqj0xhRKh7uEix99O2M0zh1we4LPamxGrVdUoALkjxjiTcXsnkk/Y4mHN2RKhmqV5sxSo1N0kGDHAb17dXN/wBMD8ucLqbkuT4qDApbSk85b/eOPAeutSjb3tYHCnSF/wCZP+LBEGfgAo94mZLq/GrL9GeNsrMx3XQ67OFiWzpCTo20nb4TewPPmMR3F7FNFYqDknMNTGYm3W9aAwvwyQq4t7n5lVvbFkpWYvxd5yLAkMVOWgEf3JoLDZ/iUVaR8ioHAc8PqtUtLlXqYQwlWvwrRuT6eZOnT9NR9FYSeVK1mLTi0f8AUwQRF2ThZVWat2aeHbktVPpuQ35E9Cgl59mW4G0X/fKiCP8A4eeAlc7MmUKaVtCOlTelIDLjRK0kCxHe2CiPbb1xbeuzZVIYMdlMdLCRYISbf0cRjXKoXi4p5Kke48ycai5D3/pWScZjG77VbE9nXJsdpzw9JUwparkx5TqTf1NlbnAOV2caSiWpyJVapEZ6tBbbn0BUm/64n2SGVvlwBOm9wU4GzpyALDZJ/exfotsrn1RpQ432f23kBSa7Wrf81n/x4WJZTLSBY2+mFhdn2t4j0r0OwWobS1samVWOyVHTfffSdvyxy+Cfebid5UZagPMsJKEd5y2JSkG3ythYWIJTWgntWcxovCDZIU2shKkLQlaTzP7wPoMNxnJLVVKHjUpcZ3QE64zUdCrXvz7q468vXCwsY4XJtF03SCZjqMzL9dp1MjTZC0SVFK333VOLHLcA+QH/AKcPuLliDBZU4tK50gpsX5h71VvQX2SPYADCwsEdMJQ+lzUaaPFyIiI0dlhC7BLTen9Mba/HTFirdaUptQHIHb7YWFiF3yiyPatZqUAJg1KI1U2FqeT5rX1J2xE1TT3Et5tJOkEjfCwsU8fpN5KYdScUiSpIOyCbY4Xll2I5q9cLCx1Ppcf7WYiQWE4WFhYBEv/Z');
INSERT INTO `users` VALUES (10002, '10002', '安珞', '$2a$10$OFdlvYKWnAqI07uVKQDvCuIvAEF/NpsF/WL8IERY4gKBpwYuRSyce', 2, '　 　静待花开　ˇ　', '#ff0000', 420, 1800, 6, 1, '2026-09-04 10:11:46.000', '2026-09-04 10:11:46.000', '2026-02-03 10:00:00.000', '2026-02-03 10:00:00.000', '1031047330.png', 1, 10001, NULL, 180, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (10003, '10003', '咏荷', '$2a$10$f9Cben3fe2myFR83ZUnUSeHTNgVo106QDcD1E68D2dhK.2.ESpKdO', 2, '咏而归，荷风送香', '#008000', 210, 900, 4, 1, NULL, NULL, '2026-02-20 11:00:00.000', '2026-02-20 11:00:00.000', '1031047330.png', 0, 0, NULL, 90, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (10004, '10004', '闲云野鹤', '$2a$10$bZCqIp/iqGh2kOk995R9VO5dfk3vwnayeh7AVDU.5a3WQx2HrymzC', 1, '宠辱不惊，看庭前花开花落', '#004299', 150, 600, 4, 1, NULL, NULL, '2026-03-15 14:00:00.000', '2026-03-15 14:00:00.000', '125703412.png', 0, 0, NULL, 60, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (10005, '10005', '蓝天', '$2a$10$XrL3Hjppc.YCMHr4T2dP5eWTL9lgKV0EjMtzEYlsBIyGG4az.Iqs.', 1, '面朝大海，春暖花开', '#800080', 98, 400, 3, 1, NULL, NULL, '2026-04-01 15:00:00.000', '2026-04-01 15:00:00.000', '104039478.jpg', 0, 0, NULL, 40, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (10006, '10006', '小忆', '$2a$10$ylFdi6kGCE.s1PCyb332nuTpYmJHQDG4I61qJgFDV3vlZivlu74km', 2, '', '', 100, 0, 1, 1, NULL, NULL, '2026-08-10 09:00:00.000', '2026-08-10 09:00:00.000', NULL, 0, 0, '', 0, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (10007, '10007', '夜凌云', '$2a$10$xGsKUcZjIcT994ZvlzSAcuAYi.aPvQ4BbjkPG0OuehG/bfWmLpuMa', 1, '待到秋来九月八', '', 13000, 130, 2, 1, '2026-09-07 16:08:49.939', '2026-09-07 16:08:49.939', '2026-08-20 09:00:00.000', '2026-08-20 09:00:00.000', '', 0, 0, NULL, 7, 0, 5, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '', 7.6, '2026-09-07', 12464, 0, 12464, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35792031, '35792031', '宋', '$2a$10$zqo.ppqaz4QiKU9ZHX3mZeGoG7D4gG0Z476VxjZuswPN4PN1/veVK', 1, '', '#808080', 200, 12600, 16, 1, NULL, NULL, '2026-03-22 19:00:00.000', '2026-08-30 13:55:21.688', '', 0, 0, '', 126, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35797804, '35797804', '　　瞿詺南　', '$2a$10$z0mrEMFcQudDy0lVmbXY.u47rBhlKYSY9bREgOHg9X.Q8qX5gl7nG', 1, '', '#ff0000', 2100, 79888, 40, 1, '2026-08-30 13:44:12.000', '2026-08-30 13:44:12.000', '2026-03-10 12:00:00.000', '2026-03-10 12:00:00.000', '104039478.jpg', 2, 0, '', 378, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35799015, '35799015', '麦', '$2a$10$WzYn3RZtBgpanBK5aBsMkemt8XkbZCh75JrPDCeuGqwerWy99HE5q', 1, '', '#BC8F8F', 200, 10900, 15, 1, NULL, NULL, '2026-04-02 16:00:00.000', '2026-08-30 13:55:21.627', '', 0, 0, '', 109, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35803370, '35803370', '刘乐乐ヾ', '$2a$10$J3D6vYNdcDnfcZui1TSCseBxxAlYYbWj6jXqfmNzn62WZtbwepvG6', 1, '', '#FF8C00', 200, 30700, 25, 1, NULL, NULL, '2026-01-10 10:00:00.000', '2026-08-30 13:55:21.566', '', 0, 0, '', 307, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35804196, '35804196', '苍笙踏歌', '$2a$10$WNh/WVOX2j738I7CR5Q08enpGi0DORVSsTJrhV7ah6igXDg4El/TS', 1, '', '#ff0000', 200, 20100, 20, 1, NULL, NULL, '2026-02-08 11:00:00.000', '2026-08-30 13:55:21.443', '', 0, 0, '', 201, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35805083, '35805083', '李春风', '$2a$10$rhMR.tTnZE8ntSi5qMjqLOhOMuiBloLJyO91mpb5uwebqOObJDj.m', 1, '', '#ff0000', 200, 44000, 30, 1, NULL, NULL, '2026-01-20 09:00:00.000', '2026-08-30 13:55:21.250', '', 0, 0, '', 440, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35805532, '35805532', '懒织红笺', '$2a$10$M6.R/6Qxxe10JW9sB5fUEO/5IxuAvY75v3jrGQbud32so434add/2', 1, '', '#ff0000', 200, 22400, 21, 1, NULL, NULL, '2026-02-16 15:00:00.000', '2026-08-30 13:55:21.504', '', 0, 0, '', 224, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35805768, '35805768', '尊上', '$2a$10$uS715fpKBmsJgmeBXf9DJeTb6hnYmdH2h6vPPdClgFEd8xIfkCQNK', 1, '', '#6495ED', 200, 53300, 33, 1, NULL, NULL, '2026-01-28 14:30:00.000', '2026-08-30 13:55:21.319', '', 0, 0, '', 533, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806066, '35806066', '断念', '$2a$10$kM4tY0qp9hPoaqyYI1zejOPISsQ96Iwr1vtRq7gmAbN43AY6NGpqa', 1, '', '#ff0000', 200, 10800, 15, 1, NULL, NULL, '2026-04-20 22:00:00.000', '2026-08-30 13:55:21.951', '', 0, 0, '', 108, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806077, '35806077', '轻轻淡写', '$2a$10$Dldc0RbZo72bBhTqA/VGXu/gNvPDRxLYPLof42JKLjmAQAzW/Zkgq', 1, '', '#ff0000', 200, 21500, 21, 1, NULL, NULL, '2026-03-18 20:00:00.000', '2026-08-30 13:55:22.011', '', 0, 0, '', 215, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806088, '35806088', '秋水未央', '$2a$10$xf9eriVWWqiBfq0p2/bWweLVqhCsDtOwfGoW/V1suNJCihzyQiTP.', 1, '', '#ff0000', 200, 38200, 28, 1, NULL, NULL, '2026-02-01 10:00:00.000', '2026-08-30 13:55:21.757', '', 0, 0, '', 382, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806114, '35806114', '   ╰┈→cc', '$2a$10$OwA0BPEQLVp1256kldm4K.lUuEprr5d71ToaJBFX/Dl.V5RYYkpTG', 1, '', '#ff0000', 200, 41500, 29, 1, NULL, NULL, '2026-01-25 13:00:00.000', '2026-08-30 13:55:21.826', '', 0, 0, '', 415, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806199, '35806199', '老王', '$2a$10$sF8V3iUB//YjIcgHF.LdIO.Oi9bdwT8M169TMQwtGQM.McHE.e8i2', 1, '', '#4169E1', 200, 30500, 25, 1, NULL, NULL, '2026-02-14 21:00:00.000', '2026-08-30 13:55:21.892', '', 0, 0, '', 305, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806205, '35806205', 'Ruby', '$2a$10$qdn54F.rz/QWPhhodL63/uuPRxoSQWxo3d9aHW6QGZ04W0USjys26', 1, '', '#ff0000', 200, 15600, 18, 1, NULL, NULL, '2026-03-05 20:00:00.000', '2026-08-30 13:55:21.381', '', 0, 0, '', 156, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806260, '35806260', '筱轩么么哒', '$2a$10$ALG8xsCIIbdH9gBrFTaoJe0EKO6ryPF2OLiNvGbvi1tGx/gAc49wu', 1, '', '#ff0000', 200, 6800, 12, 1, NULL, NULL, '2026-05-12 18:00:00.000', '2026-08-30 13:55:22.071', '', 0, 0, '', 68, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);
INSERT INTO `users` VALUES (35806345, '35806345', '胡', '$2a$10$GJ0bcCLHSzP3XMiZSpKyW.zxwx6ntXLBFjYT1qnewNmaykK0CbLUC', 1, '', '#004299', 200, 0, 1, 1, NULL, NULL, '2026-08-29 10:00:00.000', '2026-08-30 13:55:22.132', '', 0, 0, '', 0, 0, 4, 0, 1, 100, 1, 100, '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', '2026-09-04 20:08:36.000', '2026-10-04 20:08:36.000', NULL, 0, NULL, 0, 0, 0, 0, 0, 0, 0, NULL, NULL);

-- ----------------------------
-- Table structure for visitors
-- ----------------------------
DROP TABLE IF EXISTS `visitors`;
CREATE TABLE `visitors`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_id` bigint UNSIGNED NULL DEFAULT NULL,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_visitors_owner_id`(`owner_id` ASC) USING BTREE,
  INDEX `idx_visitors_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of visitors
-- ----------------------------

-- ----------------------------
-- Table structure for wallet_logs
-- ----------------------------
DROP TABLE IF EXISTS `wallet_logs`;
CREATE TABLE `wallet_logs`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `kind` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `title` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `currency` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT 'coins',
  `delta` bigint NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_wallet_logs_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 7 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of wallet_logs
-- ----------------------------
INSERT INTO `wallet_logs` VALUES (1, 10000, 'work', '打工工资', 'coins', 6, '2026-09-05 16:37:44.727');
INSERT INTO `wallet_logs` VALUES (2, 10000, 'buy', '购买「玫瑰花」×1', 'coins', -20, '2026-09-05 16:37:56.233');
INSERT INTO `wallet_logs` VALUES (3, 10001, 'sign', '每日签到', 'coins', 11, '2026-09-05 16:48:01.302');
INSERT INTO `wallet_logs` VALUES (4, 10000, 'tip', '打赏《【断念阁】新成员入阁欢迎帖》', 'coins', -100, '2026-09-05 20:05:12.655');
INSERT INTO `wallet_logs` VALUES (5, 35797804, 'tip', '收到《【断念阁】新成员入阁欢迎帖》打赏', 'coins', 100, '2026-09-05 20:05:12.660');
INSERT INTO `wallet_logs` VALUES (6, 10000, 'sign', '每日签到', 'coins', 11, '2026-09-05 20:05:29.374');

-- ----------------------------
-- Table structure for work_records
-- ----------------------------
DROP TABLE IF EXISTS `work_records`;
CREATE TABLE `work_records`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` bigint UNSIGNED NULL DEFAULT NULL,
  `created_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_work_records_user_id`(`user_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 9 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of work_records
-- ----------------------------
INSERT INTO `work_records` VALUES (1, 10001, '2026-09-04 09:51:25.117');
INSERT INTO `work_records` VALUES (2, 10007, '2026-09-04 21:42:31.987');
INSERT INTO `work_records` VALUES (3, 10007, '2026-09-04 21:42:32.861');
INSERT INTO `work_records` VALUES (4, 10007, '2026-09-04 21:42:33.319');
INSERT INTO `work_records` VALUES (5, 10001, '2026-09-05 16:30:09.211');
INSERT INTO `work_records` VALUES (6, 10001, '2026-09-05 16:30:09.652');
INSERT INTO `work_records` VALUES (7, 10001, '2026-09-05 16:30:10.497');
INSERT INTO `work_records` VALUES (8, 10000, '2026-09-05 16:37:44.718');

SET FOREIGN_KEY_CHECKS = 1;
