-- ===== 城市 68 (267,101) 恢复脚本（源：qq_jiayuan 2026-10-10 20:46 备份）=====
SET NAMES utf8mb4;
START TRANSACTION;

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_city` DISABLE KEYS */;
INSERT INTO `ezfy_city` (`id`, `user_id`, `name`, `x`, `y`, `city_level`, `feelings`, `grievance`, `tax_rate`, `pop`, `pop_max`, `gold`, `food`, `steel`, `oil`, `rare`, `gold_cap`, `food_cap`, `steel_cap`, `oil_cap`, `rare_cap`, `last_time`, `created_at`, `updated_at`, `ware_food`, `ware_steel`, `ware_oil`, `ware_rare`, `rate_food`, `rate_steel`, `rate_oil`, `rate_rare`, `placate_time`) VALUES (68,35806411,'无忧',267,101,9,50,0,50,500000,27000,2351995134,109093324,5024614499,8586071302,1304172372,54000000,2820000,6060000,5520000,5520000,1791634663165,'2026-09-26 18:03:51.868','2026-10-10 20:17:43.864',25,25,25,25,0,0,0,0,0);
/*!40000 ALTER TABLE `ezfy_city` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_city_building` DISABLE KEYS */;
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,1,9,0,0,1790417431561,'2026-09-26 18:03:51.879','2026-09-26 18:10:38.016',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418628108,'2026-09-26 18:03:51.889','2026-09-26 18:30:34.044',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,3,9,0,0,1790417956809,'2026-09-26 18:03:51.897','2026-09-26 18:19:25.728',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,19,10,0,1790420065469,1790543473214,'2026-09-26 18:04:15.810','2026-09-28 06:32:08.543',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,18,9,0,0,1790420137895,'2026-09-26 18:04:22.339','2026-09-26 18:55:38.080',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,10,11,2,1791605056510,1791684270195,'2026-09-26 18:07:36.851','2026-10-10 12:14:37.995',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,13,12,0,1791349009375,1791364600058,'2026-09-26 18:07:50.595','2026-10-07 17:16:54.980',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,21,9,0,0,1790420193719,'2026-09-26 18:08:03.267','2026-09-26 18:57:48.863',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,16,9,0,0,1790420356605,'2026-09-26 18:08:16.325','2026-09-26 18:59:18.546',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,10,0,1790420420969,1790520772391,'2026-09-26 18:08:18.644','2026-09-27 22:56:07.709',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,10,0,1790429069308,1790576269308,'2026-09-26 18:08:20.241','2026-09-28 14:46:50.009',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,10,0,1790429066003,1790576266003,'2026-09-26 18:08:21.682','2026-09-28 14:46:49.980',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,10,0,1790429068343,1790576268343,'2026-09-26 18:08:24.903','2026-09-28 14:46:49.992',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420384047,'2026-09-26 18:08:26.482','2026-09-26 18:59:57.346',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420324011,'2026-09-26 18:08:27.835','2026-09-26 18:58:44.662',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420124931,'2026-09-26 18:08:28.958','2026-09-26 18:55:27.897',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420080773,'2026-09-26 18:08:30.102','2026-09-26 18:54:49.576',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420080773,'2026-09-26 18:08:31.194','2026-09-26 18:54:49.545',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790418980688,'2026-09-26 18:08:32.323','2026-09-26 18:36:29.252',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420080773,'2026-09-26 18:08:33.680','2026-09-26 18:54:49.527',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790418843573,'2026-09-26 18:08:34.923','2026-09-26 18:34:08.089',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420080773,'2026-09-26 18:08:36.002','2026-09-26 18:54:49.508',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790418980688,'2026-09-26 18:08:38.813','2026-09-26 18:36:29.272',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,10,0,1790420422477,1790567622477,'2026-09-26 18:08:40.478','2026-09-28 12:40:33.276',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790418980688,'2026-09-26 18:08:41.852','2026-09-26 18:36:29.312',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790419304005,'2026-09-26 18:08:43.301','2026-09-26 18:41:49.094',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790419448073,'2026-09-26 18:08:45.524','2026-09-26 18:44:10.127',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,14,9,0,0,1790420080773,'2026-09-26 18:08:48.100','2026-09-26 18:54:49.469',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,7,9,0,0,1790420516458,'2026-09-26 18:08:52.881','2026-09-26 19:02:34.094',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418644040,'2026-09-26 18:08:58.184','2026-09-26 18:32:34.060',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418644040,'2026-09-26 18:08:59.823','2026-09-26 18:32:34.038',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418644040,'2026-09-26 18:09:01.386','2026-09-26 18:32:34.029',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418644040,'2026-09-26 18:09:06.291','2026-09-26 18:32:34.010',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,9,9,0,0,1790418843573,'2026-09-26 18:09:07.840','2026-09-26 18:34:08.000',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,2,9,0,0,1790418644040,'2026-09-26 18:09:09.222','2026-09-26 18:32:34.002',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:00.714','2026-09-26 18:28:28.818',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418355897,'2026-09-26 18:14:02.833','2026-09-26 18:25:56.723',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418392437,'2026-09-26 18:14:04.159','2026-09-26 18:26:33.356',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,3,9,0,0,1790418403093,'2026-09-26 18:14:06.275','2026-09-26 18:27:10.418',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:07.422','2026-09-26 18:28:28.897',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418403093,'2026-09-26 18:14:08.675','2026-09-26 18:27:10.649',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418529370,'2026-09-26 18:14:10.720','2026-09-26 18:28:49.683',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,3,9,0,0,1790418495004,'2026-09-26 18:14:12.217','2026-09-26 18:28:17.413',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:13.418','2026-09-26 18:28:28.790',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418403093,'2026-09-26 18:14:14.583','2026-09-26 18:27:10.471',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418355897,'2026-09-26 18:14:15.758','2026-09-26 18:25:56.741',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418403093,'2026-09-26 18:14:16.830','2026-09-26 18:27:10.631',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418355897,'2026-09-26 18:14:17.893','2026-09-26 18:25:56.831',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418355897,'2026-09-26 18:14:18.884','2026-09-26 18:25:56.812',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418366621,'2026-09-26 18:14:20.003','2026-09-26 18:26:09.788',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418355897,'2026-09-26 18:14:21.269','2026-09-26 18:25:56.663',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418440415,'2026-09-26 18:14:22.440','2026-09-26 18:27:34.159',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418440415,'2026-09-26 18:14:23.448','2026-09-26 18:27:34.139',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418403093,'2026-09-26 18:14:24.427','2026-09-26 18:27:10.749',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418440415,'2026-09-26 18:14:25.454','2026-09-26 18:27:34.233',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,3,9,0,0,1790418366621,'2026-09-26 18:14:26.565','2026-09-26 18:26:09.743',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,3,9,0,0,1790418403093,'2026-09-26 18:14:27.628','2026-09-26 18:27:10.440',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418403093,'2026-09-26 18:14:28.662','2026-09-26 18:27:10.490',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:29.695','2026-09-26 18:28:28.772',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418507409,'2026-09-26 18:14:30.706','2026-09-26 18:28:28.915',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418440415,'2026-09-26 18:14:31.791','2026-09-26 18:27:34.251',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418529370,'2026-09-26 18:14:32.829','2026-09-26 18:28:49.646',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418507409,'2026-09-26 18:14:33.869','2026-09-26 18:28:28.933',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:34.790','2026-09-26 18:28:28.840',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:35.814','2026-09-26 18:28:28.859',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418507409,'2026-09-26 18:14:36.723','2026-09-26 18:28:28.951',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418529370,'2026-09-26 18:14:37.816','2026-09-26 18:28:49.720',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,4,9,0,0,1790418507409,'2026-09-26 18:14:38.834','2026-09-26 18:28:28.877',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,5,9,0,0,1790418529370,'2026-09-26 18:14:39.779','2026-09-26 18:28:49.606',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,6,9,0,0,1790418529370,'2026-09-26 18:14:40.928','2026-09-26 18:28:49.763',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,17,9,0,0,1790422020322,'2026-09-26 18:59:38.090','2026-09-26 19:27:00.362',0);
INSERT INTO `ezfy_city_building` (`city_id`, `building_id`, `level`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`, `target_level`) VALUES (68,20,9,0,0,1790429790850,'2026-09-26 19:04:59.509','2026-09-26 21:36:30.948',0);
/*!40000 ALTER TABLE `ezfy_city_building` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_city_troop` DISABLE KEYS */;
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,2,532,'2026-10-09 21:52:06.680');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,3,1525804,'2026-10-10 09:58:42.943');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,4,1637489,'2026-10-09 21:37:35.637');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,6,2771691,'2026-10-09 21:32:53.828');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,7,30253361,'2026-10-10 16:40:16.021');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,8,72583419,'2026-10-10 16:42:31.445');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,9,1271278,'2026-10-10 15:06:33.155');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,10,6229127,'2026-10-09 21:37:36.195');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,11,2986087,'2026-10-10 16:40:15.897');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,13,89612,'2026-09-27 15:23:32.039');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,15,11347556,'2026-10-09 23:43:29.922');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,16,3258267,'2026-10-10 15:07:12.071');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,19,80000,'2026-10-05 10:12:08.074');
INSERT INTO `ezfy_city_troop` (`city_id`, `troop_id`, `count`, `updated_at`) VALUES (68,20,10000,'2026-10-05 10:12:08.325');
/*!40000 ALTER TABLE `ezfy_city_troop` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_city_target` DISABLE KEYS */;
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,1,0,1,0,1,'2026-10-07 23:30:17.623','2026-10-10 16:40:25.914');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,2,0,1,0,1,'2026-10-07 23:30:17.622','2026-10-10 16:40:25.885');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,3,0,1,0,-1,'2026-10-07 23:30:17.623','2026-10-10 16:40:25.895');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,4,0,1,0,1,'2026-10-07 23:30:17.631','2026-10-10 16:40:25.884');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,5,0,1,0,1,'2026-10-07 23:30:17.691','2026-10-10 16:40:25.884');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,6,0,1,0,1,'2026-10-07 23:30:17.629','2026-10-10 16:40:25.893');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,7,0,1,0,1,'2026-10-07 23:30:17.757','2026-10-10 16:40:26.083');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,8,8,0,0,1,'2026-10-07 23:30:17.761','2026-10-10 16:40:26.063');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,9,0,1,0,-1,'2026-10-07 23:30:17.757','2026-10-10 16:40:26.051');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,10,0,1,0,1,'2026-10-07 23:30:17.791','2026-10-10 16:40:26.051');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,11,0,1,0,1,'2026-10-07 23:30:17.763','2026-10-10 16:40:26.068');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,12,0,1,0,1,'2026-10-07 23:30:17.845','2026-10-10 16:40:26.079');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,13,0,1,0,1,'2026-10-07 23:30:17.888','2026-10-10 16:40:26.206');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,14,0,1,0,1,'2026-10-07 23:30:18.034','2026-10-10 16:40:26.216');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,15,0,1,0,1,'2026-10-07 23:30:17.893','2026-10-10 16:40:26.232');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,16,0,1,0,1,'2026-10-07 23:30:17.894','2026-10-10 16:40:26.225');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,17,0,0,0,0,'2026-10-07 23:30:17.997','2026-10-10 16:40:26.240');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,18,0,0,0,0,'2026-10-07 23:30:18.055','2026-10-10 16:40:26.305');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,19,0,0,0,0,'2026-10-07 23:30:17.896','2026-10-10 16:40:26.377');
INSERT INTO `ezfy_city_target` (`city_id`, `troop_id`, `atk_target_troop`, `atk_move`, `def_target_troop`, `def_move`, `created_at`, `updated_at`) VALUES (68,20,0,0,0,0,'2026-10-07 23:30:17.998','2026-10-10 16:40:26.391');
/*!40000 ALTER TABLE `ezfy_city_target` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_wildland` DISABLE KEYS */;
INSERT INTO `ezfy_wildland` (`city_id`, `x`, `y`, `wild_type`, `level`, `gain`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`) VALUES (68,269,98,2,10,'',0,0,0,'2026-10-05 10:33:12.108','2026-10-05 10:33:12.108');
INSERT INTO `ezfy_wildland` (`city_id`, `x`, `y`, `wild_type`, `level`, `gain`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`) VALUES (68,262,99,1,10,'',0,0,0,'2026-10-05 10:36:41.889','2026-10-05 10:36:41.889');
INSERT INTO `ezfy_wildland` (`city_id`, `x`, `y`, `wild_type`, `level`, `gain`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`) VALUES (68,264,102,1,10,'',0,0,0,'2026-10-05 10:36:49.713','2026-10-05 10:36:49.713');
INSERT INTO `ezfy_wildland` (`city_id`, `x`, `y`, `wild_type`, `level`, `gain`, `status`, `start_time`, `end_time`, `created_at`, `updated_at`) VALUES (68,266,102,2,10,'',0,0,0,'2026-10-09 23:38:11.110','2026-10-09 23:38:11.110');
/*!40000 ALTER TABLE `ezfy_wildland` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

/*!40000 ALTER TABLE `ezfy_equipment` DISABLE KEYS */;
INSERT INTO `ezfy_equipment` (`user_id`, `city_id`, `cfg_id`, `name`, `type`, `tier`, `military`, `logistics`, `learning`, `level`, `officer_id`, `created_at`, `slot`, `set_id`, `series`, `enhance`, `dmg`, `def`, `hp`, `move`, `crit`, `crit_dmg`, `original_user_id`) VALUES (35806411,68,13,'参谋指北针','饰品',3,30,40,40,60,2723,'2026-09-29 18:09:26.726','饰品',0,'',0,0,0,0,0,0,0,0);
INSERT INTO `ezfy_equipment` (`user_id`, `city_id`, `cfg_id`, `name`, `type`, `tier`, `military`, `logistics`, `learning`, `level`, `officer_id`, `created_at`, `slot`, `set_id`, `series`, `enhance`, `dmg`, `def`, `hp`, `move`, `crit`, `crit_dmg`, `original_user_id`) VALUES (35806411,68,11,'狙击步枪','武器',3,60,10,10,50,2723,'2026-09-29 22:52:42.915','武器',0,'',0,0,0,0,0,0,0,0);
INSERT INTO `ezfy_equipment` (`user_id`, `city_id`, `cfg_id`, `name`, `type`, `tier`, `military`, `logistics`, `learning`, `level`, `officer_id`, `created_at`, `slot`, `set_id`, `series`, `enhance`, `dmg`, `def`, `hp`, `move`, `crit`, `crit_dmg`, `original_user_id`) VALUES (35806411,68,35,'蓝宝石戒指','珠宝',1,5,5,5,1,3649,'2026-09-27 20:14:51.905','珠宝',0,'',0,0,0,0,0,0,0,0);
INSERT INTO `ezfy_equipment` (`user_id`, `city_id`, `cfg_id`, `name`, `type`, `tier`, `military`, `logistics`, `learning`, `level`, `officer_id`, `created_at`, `slot`, `set_id`, `series`, `enhance`, `dmg`, `def`, `hp`, `move`, `crit`, `crit_dmg`, `original_user_id`) VALUES (35806411,68,12,'合金装甲','防具',3,10,0,60,55,3649,'2026-09-29 18:19:25.947','防具',0,'',0,0,0,0,0,0,0,0);
INSERT INTO `ezfy_equipment` (`user_id`, `city_id`, `cfg_id`, `name`, `type`, `tier`, `military`, `logistics`, `learning`, `level`, `officer_id`, `created_at`, `slot`, `set_id`, `series`, `enhance`, `dmg`, `def`, `hp`, `move`, `crit`, `crit_dmg`, `original_user_id`) VALUES (35806411,68,30,'黑曜石戒指','珠宝',1,5,5,5,1,4817,'2026-09-28 18:50:32.870','珠宝',0,'',0,0,0,0,0,0,0,0);
/*!40000 ALTER TABLE `ezfy_equipment` ENABLE KEYS */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;


-- ---- 复活弃城时被软删除的 12 名军官 ----
UPDATE `ezfy_officer` SET `deleted_at` = NULL WHERE `city_id` = 68 AND `deleted_at` = '2026-10-12 16:16:06.367';

COMMIT;
