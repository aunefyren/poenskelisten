-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- Host: db
-- Generation Time: Nov 21, 2023 at 04:20 PM
-- Server version: 5.7.44
-- PHP Version: 8.2.12

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `db`
--

-- --------------------------------------------------------

--
-- Table structure for table `exercises`
--

CREATE TABLE `exercises` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `date` datetime(3) NOT NULL,
  `note` longtext COLLATE utf8mb4_bin,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `goal` bigint(20) NOT NULL,
  `exercise_interval` bigint(20) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- --------------------------------------------------------

--
-- Table structure for table `goals`
--

CREATE TABLE `goals` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `season` bigint(20) NOT NULL,
  `exercise_interval` bigint(20) NOT NULL DEFAULT '3',
  `competing` tinyint(1) NOT NULL DEFAULT '1',
  `user` bigint(20) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- --------------------------------------------------------

--
-- Table structure for table `groups`
--

CREATE TABLE `groups` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_bin NOT NULL,
  `description` longtext COLLATE utf8mb4_bin,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `owner_id` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `groups`
--

INSERT INTO `groups` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `enabled`, `owner_id`) VALUES`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `enabled`, `owner_id`;

-- --------------------------------------------------------

--
-- Table structure for table `group_memberships`
--

CREATE TABLE `group_memberships` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `group` bigint(20) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `member` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `group_memberships`
--

INSERT INTO `group_memberships` (`id`, `created_at`, `updated_at`, `deleted_at`, `group`, `enabled`, `member`) VALUES`id`, `created_at`, `updated_at`, `deleted_at`, `group`, `enabled`, `member`;

-- --------------------------------------------------------

--
-- Table structure for table `invites`
--

CREATE TABLE `invites` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `invite_code` varchar(191) COLLATE utf8mb4_bin NOT NULL,
  `invite_used` tinyint(1) NOT NULL DEFAULT '0',
  `invite_recipient` bigint(20) DEFAULT NULL,
  `invite_enabled` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `invites`
--

INSERT INTO `invites` (`id`, `created_at`, `updated_at`, `deleted_at`, `invite_code`, `invite_used`, `invite_recipient`, `invite_enabled`) VALUES
(1, NULL, '2022-11-20 17:22:16.505', NULL, 'TEST', 1, 1, 1),
(2, NULL, '2022-11-20 17:26:26.943', NULL, 'TEST2', 1, 2, 1),
(3, '2022-11-21 09:18:07.340', '2022-11-21 09:21:59.819', NULL, 'DCIYT4OI', 1, 3, 1),
(4, '2022-11-21 09:19:20.317', '2022-11-21 09:48:46.097', NULL, 'IC38Y159', 1, 4, 1),
(5, '2022-11-21 09:19:33.459', '2022-11-22 10:03:21.564', NULL, 'KO5XS3QM', 1, 5, 1),
(6, '2022-11-21 09:22:22.465', '2022-11-21 15:01:37.797', NULL, 'KR9NH5Z5', 1, 6, 1),
(7, '2022-11-27 00:00:00.000', '2022-12-03 17:49:47.950', NULL, 'ZZJRL6ATAD', 1, 7, 1),
(8, '2022-11-27 00:00:00.000', '2022-12-03 17:49:28.965', NULL, 'YL3VNNOW93', 1, 8, 1),
(9, '2022-11-27 00:00:00.000', '2023-11-14 19:46:33.131', NULL, '8WAD9MN12Q', 1, 10, 1),
(10, '2022-11-27 00:00:00.000', '2022-11-27 20:41:07.727', NULL, 'I49RVS6HCN', 1, 9, 1),
(11, '2022-11-27 00:00:00.000', '2022-11-27 00:00:00.000', NULL, '7MBMNB784H', 0, NULL, 1),
(12, '2022-11-27 00:00:00.000', '2022-11-27 00:00:00.000', NULL, 'ZS9MCNXTLG', 0, NULL, 1),
(13, '2022-11-27 00:00:00.000', '2022-11-27 00:00:00.000', NULL, 'N0X8RZIVDY', 0, NULL, 1),
(14, '2022-11-27 00:00:00.000', '2022-11-27 00:00:00.000', NULL, '48VC1KQEZP', 0, NULL, 1);

-- --------------------------------------------------------

--
-- Table structure for table `news`
--

CREATE TABLE `news` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `title` longtext COLLATE utf8mb4_bin NOT NULL,
  `body` longtext COLLATE utf8mb4_bin,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `date` datetime(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `news`
--

INSERT INTO `news` (`id`, `created_at`, `updated_at`, `deleted_at`, `title`, `body`, `enabled`, `date`) VALUES
(1, '2022-12-13 19:53:03.000', '2022-12-13 19:53:03.000', NULL, '🎄Get in the holiday spirit with wishlists and gift-giving', 'As Christmas approaches, the excitement in the air is palpable. The smell of peppermint and hot cocoa fills the air, and twinkling lights adorn every street corner. It\'s the most wonderful time of the year, and there\'s no better way to get into the holiday spirit than by filling out your wishlist and claiming gifts from the wishlists of your friends and loved ones.\n<br><br>\nThe beauty of a wishlist is that it allows you to be specific about the things you want, while also giving your friends and family a chance to surprise you with something thoughtful and unexpected. So as Christmas gets closer, make sure to fill out your wishlist and claim gifts from the wishlists of your friends and loved ones. It\'s a fun and easy way to spread holiday cheer and make this Christmas the best one yet. Happy holiday season!\n<br><br>\n- ChatGPT', 1, '2022-12-13 19:53:03.000');

-- --------------------------------------------------------

--
-- Table structure for table `seasons`
--

CREATE TABLE `seasons` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_bin NOT NULL,
  `description` longtext COLLATE utf8mb4_bin,
  `start` datetime(3) NOT NULL,
  `end` datetime(3) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE `users` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `first_name` longtext COLLATE utf8mb4_bin NOT NULL,
  `last_name` longtext COLLATE utf8mb4_bin NOT NULL,
  `email` varchar(191) COLLATE utf8mb4_bin NOT NULL,
  `password` longtext COLLATE utf8mb4_bin NOT NULL,
  `admin` tinyint(1) NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  `verification_code` longtext COLLATE utf8mb4_bin,
  `reset_code` longtext COLLATE utf8mb4_bin,
  `reset_expiration` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `users`
--

INSERT INTO `users` (`id`, `created_at`, `updated_at`, `deleted_at`, `first_name`, `last_name`, `email`, `password`, `admin`, `enabled`, `verified`, `verification_code`, `reset_code`, `reset_expiration`) VALUES
(1, '2022-11-20 17:22:16.501', '2023-11-18 14:11:23.571', NULL, 'Helene', 'Prestesæter Endsjø', 'helene.endsjoe@hotmail.com', '$2a$14$Z69vsFS2MK/A1z2pjIj4J.6/anNQCFQHQ.Z/nctiSgustU77q8B86', 0, 1, 1, 'KUV6Z1DG', NULL, NULL),
(2, '2022-11-20 17:26:26.940', '2023-11-01 16:25:53.065', NULL, 'Øystein', 'Aune Sverre', 'oystein.sverre@proton.me', '$2a$14$avvLXxwiQqaACFz9KCKp5u6CiYnTfORWuinxt/sEviCU.sQKzbDHC', 1, 1, 1, 'QPDLJ71G', 'IVHVJIC3', '2023-11-08 16:25:53.064'),
(3, '2022-11-21 09:21:59.816', '2022-11-21 09:21:59.816', NULL, 'Andreas', 'Sverre', 'andreas.sverre@gmail.com', '$2a$14$lswUFERIauAYqBQ13Zbu4uW6o43GTYRLgOgxPn6wKbYarwB1UqPia', 0, 1, 1, NULL, NULL, NULL),
(4, '2022-11-21 09:48:46.094', '2022-11-21 09:48:46.094', NULL, 'Beate', 'Aune', 'beate.sverre@gmail.com', '$2a$14$gq1eBmtZjTIMjWcQx5XM0OhgljXDDvUnxum5FB9i2NkcTOdoUf0vm', 0, 1, 1, NULL, NULL, NULL),
(5, '2022-11-21 15:01:37.791', '2022-11-21 15:01:37.791', NULL, 'Aurora', 'Prestesæter', 'aurpre2006@hotmail.com', '$2a$14$VL6lvljzNbWWDPIjNa61PO3ytfDwwAO9qXSx9YugX/yl/AZQb4iqu', 0, 1, 1, NULL, NULL, NULL),
(6, '2022-11-22 10:03:21.556', '2023-05-22 09:20:37.924', NULL, 'Anette', 'Endsjø', 'anette.endsjoe@gmail.com', '$2a$14$OhpLC/TBQj17t01h1a0Sae2YSG9LNZbtbna.4I72TuL.ULL0ro.o2', 0, 1, 1, NULL, 'JOGQLJFM', '2023-05-29 09:20:37.922'),
(7, '2022-11-27 20:41:07.677', '2023-11-15 09:57:52.764', NULL, 'Kristine', 'Endsjø', 'kristine-endsjoe@hotmail.com', '$2a$14$rl62jK4I3G0j0bWN2gzn.u8QJidGO57t2zQt1U28sEvqgeLngZCTy', 0, 1, 1, NULL, '2XGAONFT', '2023-11-22 09:57:52.762'),
(8, '2022-12-03 17:49:28.952', '2022-12-03 17:49:28.952', NULL, 'Linnea', 'Prestesæter', 'linpre2003@hotmail.com', '$2a$14$erZLsDUXSfNBZcigK53.Bu32879Wqqyto8z3otouIJ9eaQX/./YN6', 0, 1, 1, NULL, NULL, NULL),
(9, '2022-12-03 17:49:47.944', '2023-08-21 23:03:58.030', NULL, 'Karianne', 'Prestesæter', 'karianne@formatic.no', '$2a$14$8Ivb.oXUjkGvJPO.HJ/x0ep2xNqDTzSG0m/RabqauOus/z2.0140K', 0, 1, 1, NULL, 'RXHLK4LN', '2023-08-28 23:03:58.028'),
(10, '2023-11-14 19:46:33.113', '2023-11-14 19:47:09.985', NULL, 'Eirik', 'Soløy', 'Eirik.soloy@hotmail.com', '$2a$14$n7N.lK/FJFP5UmhCERNw6.ATqIkEK4girS6mVhXEOJQgETZfDOKDm', 0, 1, 1, 'PAOVEC8S', 'KQOYSE5E', '2023-11-14 19:46:32.059');

-- --------------------------------------------------------

--
-- Table structure for table `wishes`
--

CREATE TABLE `wishes` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_bin NOT NULL,
  `note` longtext COLLATE utf8mb4_bin,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `owner_id` bigint(20) NOT NULL,
  `url` longtext COLLATE utf8mb4_bin,
  `wishlist_id` bigint(20) NOT NULL,
  `price` double DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `wishes`
--

INSERT INTO `wishes` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `note`, `enabled`, `owner_id`, `url`, `wishlist_id`, `price`) VALUES
(1, '2022-11-20 17:24:45.687', '2022-11-26 21:03:33.204', NULL, 'Brunt lysfat fra Kremmerhuset', 'Såååå fint <3 ', 0, 1, 'https://kremmerhuset.no/interior/lysfat/lysfat-speil-40-cm-brunt-brun', 1, NULL),
(2, '2022-11-20 23:42:57.587', '2022-11-21 10:00:05.440', NULL, 'Aeotec Z-Stick 7 Plus', 'Må nok bestilles fra Amazon', 0, 2, 'https://www.amazon.com/gp/aw/d/B094NW5B68', 2, NULL),
(3, '2022-11-20 23:47:24.453', '2022-11-20 23:47:24.453', NULL, 'Sokker', '43-46', 1, 2, 'https://www.zalando.no/tommy-hilfiger-men-sock-6-pack-sokker-dark-navy-to182f04b-k11.html', 2, NULL),
(4, '2022-11-20 23:50:02.658', '2022-11-21 10:00:07.627', NULL, 'Støydempende hodetelefoner ', '', 0, 2, 'https://www.komplett.no/product/1192605/tv-lyd-bilde/hodetelefoner-tilbehoer/hodetelefoner-oerepropper/bose-qc-45-traadloese-hodetelefoner-over-ear-sort', 2, NULL),
(5, '2022-11-21 09:27:48.371', '2022-11-21 09:27:48.371', NULL, 'Mummi-kopp', 'Stinky er kul', 1, 2, 'https://www.kitchn.no/borddekking/kopper-og-krus/mummikopper/moomin-by-arabia-mummikopp-30-cl-stinky-i-aksjon/', 3, NULL),
(6, '2022-11-21 09:29:08.948', '2022-11-21 09:29:08.948', NULL, 'Mummi-kopp (Skummel)', 'Hufsa er skummel', 1, 2, 'https://www.kitchn.no/borddekking/kopper-og-krus/mummikopper/moomin-by-arabia-mummikopp-30-cl-hufsa/', 3, NULL),
(7, '2022-11-21 09:36:15.113', '2022-11-21 09:36:15.113', NULL, 'Airfrier ', 'Tilbud denne uka', 1, 3, 'https://www.power.no/kjoekkenmaskiner/kjoekkenapparater/airfryers-og-frityrkokere/airfryers/cosori-cp158-af-rxb-airfryer-svart/p-1805626/?utm_source=prisjakt&utm_medium=cpc', 4, NULL),
(8, '2022-11-21 09:36:36.125', '2022-11-21 09:36:36.125', NULL, 'Bestilkk', 'Tilbud denne uka', 1, 3, 'https://www.cg.no/christiania-bestikk-milano-bestikksett-43-deler', 4, NULL),
(9, '2022-11-21 09:46:40.916', '2022-11-21 09:46:40.916', NULL, 'Dyr hylle', 'Spleis kanskje?', 1, 2, 'https://lawadesign.dk/products/twist-shelf', 3, NULL),
(10, '2022-11-22 10:10:17.681', '2023-05-22 09:21:00.216', NULL, 'Helena Rubinstein Lash Queen Mascara - Perfect Blacks ', 'No. 0000001 - Perfect Black', 0, 6, 'https://www.tax-free.no/no/product1149902/helena-rubinstein-lash-queen-perfect-blacks', 5, NULL),
(11, '2022-11-22 10:11:55.893', '2023-10-22 20:53:04.740', NULL, 'Villeroy boch servise - Anmut gold', '8 x 27 cm tallerken, og 12 × 24 cm dyp tallerken', 1, 6, '', 5, 0),
(12, '2022-11-22 10:17:33.080', '2023-05-22 09:21:04.968', NULL, 'Eleni og Chris - Intensive Treatment Oil ', '(fås hos Sayso) ', 0, 6, 'https://eleniandchris.no/collections/hair-styling/products/ec-intensive-treatment-oil-100ml', 5, NULL),
(13, '2022-11-22 10:46:23.448', '2023-05-22 09:29:26.329', NULL, 'Joggebukse ', 'Ala den Adidas-buksa jeg har, gjerne litt bred kant i livet ', 0, 6, '', 5, NULL),
(14, '2022-11-22 21:06:17.613', '2022-11-22 21:06:17.613', NULL, 'Click and Grow Smart Garden 3 ', 'Planteeeeeeer', 1, 2, 'https://www.kjell.com/no/produkter/hjem-fritid/hage/selvvannende-krukker/click-and-grow-smart-garden-3-startpakke-hvit-p47105', 3, NULL),
(15, '2022-11-23 11:17:36.286', '2023-06-19 16:05:56.881', NULL, 'Penger til kandidatringen', '', 0, 1, '', 1, NULL),
(16, '2022-11-23 11:20:29.147', '2023-11-18 14:06:40.946', NULL, 'Headset fra Sony', 'For å stenge ut bassen fra naboen når Helene leser skole :)))', 0, 1, 'https://www.prisjakt.no/product.php?p=5250941&utm_campaign=alerts&utm_medium=email&utm_source=alert', 1, NULL),
(17, '2022-11-23 11:20:54.277', '2023-06-19 16:06:01.201', NULL, 'Billett/gavekort til The Well', 'Needs to relax', 0, 1, '', 1, NULL),
(18, '2022-11-23 11:22:06.640', '2023-10-07 10:47:20.769', NULL, 'Vaser fra Magnor', 'Helst blå/koks-farger. Ikke klart glass! Seriene Rocks, Iglo og Boblen <3 KAN KJØPES BRUKT!', 0, 1, '', 1, NULL),
(19, '2022-11-23 11:22:30.821', '2022-11-26 21:03:42.420', NULL, 'Wool kurv fra Kid', 'So cute <3 ', 0, 1, 'https://www.kid.no/jul/jul-pa-kjokkenet/wool-kurv-hvit?v=208085025000', 1, NULL),
(20, '2022-11-23 11:23:51.360', '2023-06-19 16:06:36.860', NULL, 'It\'s fine - Taz Alam', 'Wow! Helene ønsker seg en bok???', 0, 1, 'https://www.adlibris.com/no/bok/its-fine-its-fine-its-fine-9780008501389', 1, NULL),
(21, '2022-11-23 11:25:21.123', '2023-06-19 16:06:41.701', NULL, 'Gin-glass', 'Sånne fine med mønster i glasset. Kan sikkert finne link etterhvert :)', 0, 1, '', 1, NULL),
(22, '2022-11-24 08:29:08.275', '2022-11-24 08:29:11.195', NULL, 'testo', 'test', 0, 2, '', 2, NULL),
(23, '2022-11-26 14:04:02.910', '2022-11-26 14:04:02.910', NULL, 'Knaggrekke', '', 1, 3, 'https://www2.hm.com/no_no/productpage.0890058002.html', 4, NULL),
(24, '2022-11-26 19:34:41.099', '2022-11-26 19:34:41.099', NULL, 'Rumpe 1', '', 1, 3, 'https://www.ellos.no/byon/vase-butt/1611824-01-0?utm_campaign=paid-search_shopping&gclid=Cj0KCQiAj4ecBhD3ARIsAM4Q_jGOip2hWQiZPtEoTcELdnwcRlydXqn0vY0UQK4kkx8n_NZxDr2ZXoUaAgtrEALw_wcB&gclsrc=aw.ds', 4, NULL),
(25, '2022-11-26 19:35:12.139', '2022-11-26 19:35:12.139', NULL, 'Rumpe 2', '', 1, 3, 'https://www.ellos.no/byon/vase-nature-19-cm/1573100-01-0?utm_campaign=paid-search_shopping&gclid=Cj0KCQiAj4ecBhD3ARIsAM4Q_jH9Jye0HoY_s5HOqLj9gIQlcFIWiowf4kYrSAK4p69EeDuT_QswyR0aAsWgEALw_wcB&gclsrc=aw.ds', 4, NULL),
(26, '2022-11-26 20:58:28.462', '2022-11-26 20:58:28.462', NULL, 'Lysfat', '', 1, 2, 'https://kremmerhuset.no/interior/lysfat/lysfat-speil-40-cm-brunt-brun', 3, NULL),
(27, '2022-11-26 21:00:14.970', '2022-11-26 21:00:14.970', NULL, 'Vaser fra Magnor', 'Helst blå/koks-farger. Ikke klart glass! Seriene Rocks, Iglo og Boblen <3 KAN KJØPES BRUKT!', 1, 2, '', 3, NULL),
(28, '2022-11-26 21:02:14.990', '2022-11-26 21:02:14.990', NULL, 'Wool kurv', '', 1, 2, 'https://www.kid.no/jul/jul-pa-kjokkenet/wool-kurv-hvit?v=208085025000', 3, NULL),
(29, '2022-11-26 21:06:26.568', '2022-11-26 21:06:26.568', NULL, 'Server kabinett', '', 1, 2, 'https://www.multicom.no/silverstone-case-storage-cs380-v2/cat-p/c/p1000507402', 7, NULL),
(30, '2022-11-26 21:07:12.661', '2022-11-26 21:07:12.661', NULL, 'CPU vifte', '', 1, 2, 'https://www.komplett.no/product/815299/datautstyr/pc-komponenter/vifterkjoelingvannkjoeling/cpu-luftkjoeling/noctua-nh-d15-cpu-kjoeler', 7, NULL),
(31, '2022-11-26 21:07:55.169', '2022-11-26 21:07:55.169', NULL, 'Optikertime', 'Jeg ser ingenting ', 1, 2, '', 7, NULL),
(32, '2022-11-26 21:08:36.753', '2022-11-26 21:08:36.753', NULL, 'Sokker', '', 1, 2, 'https://www.zalando.no/tommy-hilfiger-men-sock-6-pack-sokker-black-to182f04b-q11.html', 7, NULL),
(33, '2022-11-26 21:10:03.315', '2022-11-26 21:10:03.315', NULL, 'Z-Wave USB', '', 1, 2, 'https://www.amazon.com/gp/aw/d/B094NW5B68', 7, NULL),
(34, '2022-11-26 21:13:37.880', '2022-11-26 21:13:37.880', NULL, 'Lysdimmer', 'Trenger 9 stykker faktisk ', 1, 2, 'https://www.kjell.com/no/produkter/smarte-hjem/fjernkontroller/smarte-dimmerer/heatit-z-dim-2-z-wave-multidimmer-250-w-p52116', 7, NULL),
(35, '2022-11-27 17:26:59.939', '2023-06-19 16:06:45.106', NULL, 'Alt fra denne ønskelisten på Kremmerhuset', 'https://kremmerhuset.no/wishlist/paswabrucihom', 0, 1, 'https://kremmerhuset.no/wishlist/paswabrucihom', 1, NULL),
(36, '2022-11-27 17:37:19.538', '2023-08-15 13:12:07.575', NULL, 'The INKEY List Caffeine Eye Cream', 'så trøtt :((', 0, 1, 'https://www.theinkeylist.com/products/caffeine-eye-cream', 1, NULL),
(37, '2022-11-27 20:44:48.785', '2022-11-27 20:44:53.617', NULL, 'æøåæøå', 'æøå', 0, 2, '', 7, NULL),
(38, '2022-11-27 21:01:10.113', '2023-05-29 17:29:00.416', NULL, 'Modern House Soft Grey Gin glass ', '', 0, 6, 'https://www.kitchn.no/borddekking/glass/cocktailglass/modern-house-soft-grey-ginglass-60-cl-4-stk-roykgra-gull/', 5, NULL),
(39, '2022-11-28 09:16:24.744', '2023-08-15 13:12:18.279', NULL, 'NOLA Sofabord', '', 0, 1, 'https://www.bohus.no/stue/bord/sofabord/nola-sofabord-b-120-1', 1, NULL),
(40, '2022-11-28 09:17:14.642', '2023-06-19 16:07:24.910', NULL, 'London Blue Topaz Ring - Ilse', 'Str. 8 ', 0, 1, 'https://www.linjer.co/collections/rings/products/london-blue-topaz-ring-ilse', 1, NULL),
(41, '2022-11-28 09:18:45.453', '2023-08-15 13:12:27.827', NULL, 'Open Leaf Ring - Ada', 'Str. 8', 0, 1, 'https://www.linjer.co/collections/rings/products/open-leaf-ring-ada', 1, NULL),
(42, '2022-11-28 09:20:24.535', '2022-11-28 09:20:24.535', NULL, 'Pearl Drop Earrings - Mathilde', '', 1, 1, 'https://www.linjer.co/collections/pearls/products/pearl-drop-earrings-mathilde', 1, NULL),
(43, '2022-11-28 09:21:08.279', '2022-11-28 09:21:08.279', NULL, 'Keshi Pearl Necklace - Marit', '', 1, 1, 'https://www.linjer.co/collections/pearls/products/keshi-pearl-necklace-marit', 1, NULL),
(44, '2022-11-28 09:21:30.381', '2023-11-05 16:35:11.340', NULL, 'Hoop Earrings with Pearl - Rebecca', '', 0, 1, 'https://www.linjer.co/collections/pearls/products/hoop-earrings-with-pearl-rebecca', 1, NULL),
(45, '2022-11-28 12:29:44.844', '2023-05-22 09:21:13.366', NULL, 'Rifla ostehøvel ', '', 0, 6, '', 5, NULL),
(46, '2022-11-28 21:22:23.824', '2022-11-28 21:22:23.824', NULL, 'Hot Fuzz 4K Blu-Ray', '', 1, 2, 'https://www.zavvi.com/4k/hot-fuzz-zavvi-exclusive-limited-edition-4k-ultra-hd-steelbook-includes-blu-ray/14197029.html', 7, NULL),
(47, '2022-12-07 09:15:54.749', '2023-05-22 09:21:16.906', NULL, 'Ulvang Rav Genser ', 'Str L (?), i veldig nøytrale farger, lys grå eller hvit ', 0, 6, '', 5, NULL),
(48, '2022-12-08 12:33:27.580', '2023-05-22 09:29:52.509', NULL, 'Sulten, av Jan Ivar Nykvist ', 'Jeg kan ønske meg bøker nå! Fordi Linnea også får rabatt! :D ', 0, 6, '', 5, NULL),
(49, '2022-12-08 12:34:37.845', '2023-05-22 09:21:21.315', NULL, 'Harry Potter and the Order of the Phoenix - illustrert (den nye illustrerte på engelsk!) ', 'Jeg kan ønske meg bøker nå fordi Linnea jobber på Ark og også får rabatt! :D ', 0, 6, '', 5, NULL),
(50, '2022-12-11 10:36:21.920', '2022-12-11 10:36:21.920', NULL, 'Tynt ulltøy (typ Cubus)', '', 1, 9, '', 8, NULL),
(51, '2022-12-11 10:36:29.346', '2022-12-11 10:36:29.346', NULL, 'Tynne ullsokker', '', 1, 9, '', 8, NULL),
(52, '2022-12-11 10:36:52.650', '2022-12-11 10:36:52.650', NULL, 'Stavmixer (typ Grundig)', '', 1, 9, '', 8, NULL),
(53, '2022-12-12 07:23:43.738', '2022-12-12 07:23:43.738', NULL, 'Bowling med hele gjengen', '', 1, 9, '', 8, NULL),
(54, '2022-12-14 18:24:50.957', '2022-12-14 18:24:50.957', NULL, 'led lys', 'emh sånne strips… se url', 1, 5, 'https://www.clasohlson.com/no/LED-list-RGBW-utbyggbar-med-fjernkontroll,-Cotech/p/36-7974', 9, NULL),
(55, '2022-12-14 18:30:51.844', '2022-12-14 18:30:51.844', NULL, 'shaving høvler', 'de finnes på normal', 1, 5, 'https://gillettevenus.co.uk/en-gb/shaving-products/womens-razors/venus-comfortglide-spa-breeze-razor/', 9, NULL),
(56, '2022-12-14 18:35:19.823', '2022-12-14 18:35:19.823', NULL, 'stjernelys', 'yeee kan hende det finnes andre steder en clas også', 1, 5, 'https://www.clasohlson.com/no/p/36-8284?utm_source=google&utm_medium=organic&utm_campaign=google%20surfaces&gclid=CjwKCAiAheacBhB8EiwAItVO22V7RndjCppbh9LNyzpwzUOwp2gq8Yy7p2YMMCpmS_cYPWNDyvukRhoCR_AQAvD_BwE', 9, NULL),
(57, '2022-12-14 18:39:54.272', '2022-12-14 18:39:54.272', NULL, 'duftlys', 'helst en litt mild milk, honey, vanilla eller no sånt.. ikke blomst', 1, 5, 'https://lyko.com/no/country-candle/country-candle-daylight-vanilla-cupcake?gclid=CjwKCAiAheacBhB8EiwAItVO23U-uqjnJ32OLBtf8TPW1n7_6uZRTV51kJ1mX1TcCGuMBE7GNOEScRoCkewQAvD_BwE', 9, NULL),
(58, '2022-12-14 18:44:29.342', '2022-12-14 18:44:29.342', NULL, 'vippeserum', '', 1, 5, 'https://xlash.com/no/xlash-eyelash-serum-1ml?gclid=CjwKCAiAheacBhB8EiwAItVO21ENqMn4PbyZsf5u5ER1rLJBBPWD18pmiEmcwUKXFVmbFa5CA64E9hoClB4QAvD_BwE', 9, NULL),
(59, '2023-02-06 11:18:11.117', '2023-06-03 20:08:01.021', NULL, 'Bluesound PULSE M (P230) trådløs høyttaler', 'Kan spleises på?', 0, 2, 'https://www.hifiklubben.no/bluesound-pulse-m-p230-traadloes-hoeyttaler/blsp230bk/', 10, 5998),
(60, '2023-02-06 11:43:53.140', '2023-05-13 14:32:45.162', NULL, 'Bose QC 45 trådløse hodetelefoner', 'Må nok spleises på', 1, 2, 'https://www.komplett.no/product/1192605/tv-lyd-bilde/hodetelefoner-tilbehoer/hodetelefoner-oerepropper/bose-qc-45-traadloese-hodetelefoner-over-ear-sort#', 10, 3990),
(61, '2023-02-06 11:51:59.477', '2023-06-24 23:32:03.333', NULL, 'Hot Fuzz - 4K Ultra HD Blu-Ray', 'Kan bestilles via lenken', 1, 2, 'https://www.zavvi.com/4k/hot-fuzz-zavvi-exclusive-limited-edition-4k-ultra-hd-steelbook-includes-blu-ray/14197029.html', 10, 350),
(62, '2023-02-06 11:52:19.481', '2023-05-13 14:32:13.914', NULL, 'Svarte sokker', 'Trengs alltid', 1, 2, 'https://www.zalando.no/tommy-hilfiger-men-sock-6-pack-sokker-black-to182f04b-q11.html', 10, 439),
(63, '2023-02-06 11:53:08.498', '2023-05-13 14:32:03.859', NULL, 'Aeotec Z-Stick 7', '', 1, 2, 'https://www.komplett.no/product/1206255/hjem-fritid/smarte-hjem/hjemmesentraler/aeotec-z-stick-7', 10, 599),
(64, '2023-02-06 11:58:28.156', '2023-05-16 12:38:04.343', NULL, 'Multisensor', 'For hjemme-automasjon', 0, 2, 'https://www.komplett.no/product/1206244/hjem-fritid/smarte-hjem/alarm-sikkerhet/bevegelsessensorer/aeotec-multisensor-7-hvit', 10, NULL),
(65, '2023-02-06 12:00:41.058', '2023-05-13 14:31:17.355', NULL, 'Smart-plugg', 'Ikke ofte på lager', 1, 2, 'https://www.elektroimportoren.no/?Article=1407927', 10, 699),
(66, '2023-05-02 11:58:20.012', '2023-05-13 14:30:50.634', NULL, 'WILFA WSFBS-200B KAFFEKVERN', '', 1, 2, 'https://www.power.no/kjoekkenmaskiner/kaffe-og-te/kaffekvern/wilfa-wsfbs-200b-kaffekvern/p-1809325/', 10, 4199),
(67, '2023-05-02 12:12:31.963', '2023-05-02 12:12:34.864', NULL, 'Tester', '', 0, 2, '', 10, NULL),
(68, '2023-05-16 12:44:49.088', '2023-05-16 12:44:49.088', NULL, 'SAMSUNG WIRELESS CHARGER PAD TRÅDLØS LADER', '', 1, 2, 'https://www.power.no/mobil-og-foto/ladere-og-kabler/traadloes-lader/samsung-wireless-charger-pad-traadloes-lader/p-1256559/', 10, 599),
(69, '2023-05-22 09:21:54.363', '2023-05-29 17:29:03.178', NULL, 'Tursekk til dagsturer (både ski og gåtur)', 'Plass til et par flasker, litt klær og mat', 0, 6, '', 5, 0),
(70, '2023-05-22 09:24:16.303', '2023-05-22 09:24:16.303', NULL, 'Gavekort Kitchn eller tilbords ', 'Ønsker mer vinglass etc, men vet ikke hvilke enda', 1, 6, '', 5, 0),
(71, '2023-05-22 09:27:13.994', '2023-05-22 09:27:13.994', NULL, 'Clinique Almost lipstick 06 Black Honey', '', 1, 6, '', 5, 0),
(72, '2023-05-22 09:29:19.777', '2023-08-16 10:45:06.275', NULL, 'Kjøleveske grønn 14L fra Clas Ohlson', '', 0, 6, 'https://www.clasohlson.com/no/Kj%C3%B8leveske,-14-liter/p/31-6229-6?gclid=CjwKCAjwpayjBhAnEiwA-7ena8FBe6fadr5ksrAppxqhqitc1P2ql3zXGAG1JjPUxh3yO5LRB_zfFBoC7rYQAvD_BwE', 5, 0),
(73, '2023-05-22 23:25:58.096', '2023-05-22 23:25:58.096', NULL, 'penger', '', 1, 5, '', 12, 0),
(74, '2023-06-08 10:11:39.309', '2023-06-19 16:03:08.946', NULL, 'Pizzaspade', '', 0, 2, 'https://www.kitchn.no/kjokken/kjokkenutstyr/pizzaspader/kamado-sumo-pizzaspade-55x355-cm-akasie/', 10, 349),
(75, '2023-06-19 16:08:28.599', '2023-11-18 14:06:33.691', NULL, 'Penger til padel racket 🎾', 'Jeg er pro, need utstyr. Dette er fortsatt aktuelt siden jeg var fattig ved bursdagen min :D', 1, 1, '', 1, 2000),
(76, '2023-06-22 23:29:25.693', '2023-11-01 07:30:30.151', NULL, 'Smart home IR fjernkontroll', '', 0, 2, 'https://www.computersalg.no/i/6182980/broadlink-rm4-pro-universell-fjernkontrollmottaker-ir-rf-wi-fi-sort', 10, 1050),
(77, '2023-07-05 11:15:45.250', '2023-08-26 20:18:00.161', NULL, 'Amazon Kindle Oasis (2019)', '', 0, 2, 'https://www.komplett.no/product/1221815/pc-nettbrett/nettbrett-ipad-lesebrett/lesebrett/amazon-kindle-oasis-2019-7-32gb', 10, 3990),
(78, '2023-07-27 12:12:45.929', '2023-07-27 12:12:45.929', NULL, 'SATA PCIe Card', '', 1, 2, 'https://www.dustin.no/product/5011307186', 10, 645),
(79, '2023-08-14 22:02:26.650', '2023-08-14 22:02:26.650', NULL, 'Kyzar Joy-Con Ladestasjon Switch', '', 1, 2, 'https://www.komplett.no/product/1171196/gaming/tilbehoer-til-spillkonsoller/kyzar-joy-con-ladestasjon-switch', 10, 329),
(80, '2023-08-14 22:03:56.089', '2023-10-13 08:21:30.277', NULL, 'Nintendo Switch Joy-Con (Neon Red/Neon Blue)', '', 0, 2, 'https://www.prisjakt.no/product.php?p=4111408', 10, 799),
(81, '2023-08-14 22:04:25.482', '2023-08-29 15:17:56.570', NULL, 'Nintendo Switch Joy-Con (Pastel Pink/Pastel Yellow)', '', 1, 2, 'https://www.prisjakt.no/product.php?p=11110382', 10, 899),
(82, '2023-08-15 13:15:21.973', '2023-11-12 14:30:31.177', NULL, 'Dekkebrikke 🍽️', 'Raw Organic Dekkebrikke, cinnamon ✨ Trenger til sammen 3 stk ❤️', 0, 1, 'https://www.kitchn.no/borddekking/serveringstilbehor/bordbrikker/aida-raw-recycled-dekkebrikke-41x335-cm-cinnamon/', 1, 119),
(83, '2023-08-16 10:46:39.115', '2023-09-16 16:11:13.945', NULL, 'Magnor noir vinkaraffel (Halvor Bakke serien)', '', 1, 6, 'https://www.tilbords.no/borddekking/flasker-og-kanner/vannkarafler/magnor-noir-karaffel-108l/', 5, 0),
(84, '2023-08-17 14:07:35.961', '2023-08-17 14:07:35.961', NULL, 'Vindmakeren av Lisa Aisato og Maja Lunde', '', 1, 6, '', 5, 0),
(85, '2023-08-17 14:08:15.007', '2023-08-17 14:08:15.007', NULL, 'Drømmen om et tre, av Maja Lunde i pocket (MÅ VÆRE POCKET! skal matche på hylla)', '', 1, 6, '', 5, 0),
(86, '2023-08-17 14:08:42.697', '2023-08-17 14:08:42.697', NULL, 'Faen, faen, faen, av Linn Strømberg. Innbundet', '', 1, 6, '', 5, 0),
(87, '2023-10-07 10:52:15.561', '2023-10-07 10:52:15.561', NULL, 'Dunpute', 'Sånn skikkelig god dunpute. Halvhøy-høy ❤️', 1, 1, '', 1, 0),
(88, '2023-10-22 15:05:45.938', '2023-10-22 15:05:45.938', NULL, 'Ladeplate til iPhone (og kanskje Apple Watch)', 'Modell kommer snart', 1, 1, '', 1, 0),
(89, '2023-10-25 21:13:11.295', '2023-10-25 21:13:11.295', NULL, 'Skøyter', 'Danseskøyter. Str 39/40 ⛸️', 1, 1, '', 1, 0),
(90, '2023-11-01 07:29:47.083', '2023-11-01 07:29:47.083', NULL, 'Lego The Office', 'Ikke døm meg', 1, 2, 'https://lekekassen.no/lego-ideas-21336-the-office', 10, 1250),
(91, '2023-11-02 23:49:16.607', '2023-11-05 13:08:46.813', NULL, 'SODASTREAM DUO KULLSYREMASKIN, SVART', '', 0, 2, 'https://www.power.no/kjoekkenmaskiner/vann-og-juice/kullsyremaskin/sodastream-duo-kullsyremaskin-svart/p-1337464/', 13, 1299),
(92, '2023-11-05 12:38:22.759', '2023-11-05 12:38:22.759', NULL, 'Click & grow Potte', 'Eller liknende', 1, 1, 'https://www.obs.no/bygg-og-hage/hage-og-uterom/fro-og-dyrking/2098370?v=Obs-4742793007205', 13, 1000),
(93, '2023-11-05 12:43:59.895', '2023-11-05 12:43:59.895', NULL, 'Twist shelf - LAWA design', '', 1, 1, 'https://lawadesign.dk/products/twist-shelf', 13, 3400),
(94, '2023-11-05 12:46:59.706', '2023-11-05 12:46:59.706', NULL, 'Stor kjevle av tre', '', 1, 1, '', 13, 0),
(95, '2023-11-07 08:12:16.936', '2023-11-07 08:12:16.936', NULL, 'Karbonstål stekepanne', '', 1, 2, 'https://www.jernia.no/kj%C3%B8kkenutstyr/stekepanner/stekepanne/beka-stekepanne-nomad-karbonst%C3%A5l-28cm/p/58019341', 13, 799),
(96, '2023-11-12 20:41:50.168', '2023-11-12 20:41:50.168', NULL, 'Secret Hitler brettspill', '', 1, 2, 'https://gamezone.no/brettspill/136242/secret-hitler-kortspill', 13, 600),
(97, '2023-11-13 08:49:07.208', '2023-11-13 08:49:07.208', NULL, 'Oppenheimer (2023) 4K Blu-ray steelbook', '', 1, 2, 'https://cdon.no/film/oppenheimer-limited-steelbook-4k-ultra-hd-blu-ray-cdon-exclusive-140026724', 10, 350),
(98, '2023-11-15 10:08:26.914', '2023-11-15 10:08:26.914', NULL, 'Årsmedlemsskap nasjonalmuseet (Kristine)', '', 1, 7, 'https://www.nasjonalmuseet.no/om-nasjonalmuseet/bli-medlem/medlemmer/', 14, 700),
(99, '2023-11-15 10:10:54.874', '2023-11-15 10:10:54.874', NULL, 'The Ordinary Buffet + copper peptides serum (K) ', '', 1, 7, '', 14, 400),
(100, '2023-11-15 17:45:19.027', '2023-11-15 17:45:34.052', NULL, 'Buff til løping (se link) (K)', 'Landet på at denne er best (merino=ikke like mye lukt/bruke flere ganger, men ikke like vindtett (?)) + pustehull og ser ut som den sitter godt, åpen for alternativer, men veldig i denne gata', 1, 7, 'https://www.fjellsport.no/herreklaer/hals/aclima-doublewool-neckgaiter-unisex-jet-black', 14, 500),
(101, '2023-11-15 18:04:18.051', '2023-11-15 18:06:01.709', NULL, 'Gamle/arve mellomtykke ullsokker (K)', 'Ønsker meg gamle (arve) mellomtykke ullsokker til å løpe i, altså ikke nye! ', 0, 7, '', 14, 0),
(102, '2023-11-15 18:06:00.202', '2023-11-15 18:06:00.202', NULL, 'Ullsokker til hverdag', 'Helt standard svarte ullsokker (typ Devold/Ulvang). Str. 38-39. ', 1, 1, '', 1, 0),
(103, '2023-11-15 18:38:21.821', '2023-11-15 18:38:21.821', NULL, 'Kakeform med lokk', 'MÅ ikke være den i linken, kan være hva som helst så lenge det er lokk :)', 1, 1, 'https://www.tilbords.no/kjokken/bakeutstyr/brodformer-og-bakeformer/modern-house-bayk-kakeform-m-lokk-38x25-cm-35l-karbonstal/', 1, 200),
(104, '2023-11-15 18:38:50.302', '2023-11-15 18:38:50.302', NULL, 'Kakespade', '', 1, 1, 'https://www.tilbords.no/borddekking/bestikk/serveringsbestikk/aida-raw-kakespade-40-cm-svart/', 13, 100),
(105, '2023-11-15 18:39:13.313', '2023-11-15 18:39:13.313', NULL, 'Kakeserveringssett', '', 1, 1, 'https://www.tilbords.no/borddekking/bestikk/serveringsbestikk/modern-house-gold-serveringssett-2-deler-satin/', 13, 430),
(106, '2023-11-15 18:45:34.568', '2023-11-15 18:45:34.568', NULL, 'Potetskreller', '', 1, 1, '', 13, 0),
(107, '2023-11-15 18:46:00.038', '2023-11-15 18:46:00.038', NULL, 'Sitruspresse', '', 1, 1, 'https://www.tilbords.no/kjokken/kjokkenutstyr/sitruspresser/cilio-limonta-sitronpresse/', 13, 190),
(108, '2023-11-16 18:37:13.201', '2023-11-16 18:37:13.201', NULL, 'Keramikk-kurs ❤️', 'Dyrt, men kult ✨', 1, 1, '', 1, 0),
(109, '2023-11-16 18:52:50.536', '2023-11-16 18:52:50.536', NULL, 'Osteklokke', '', 1, 1, 'https://bakerenogkokken.no/flere-produkter/oppbevaring/rig-tig-contain-it-osteklokkeoppbevaringsboks-svart/', 13, 400),
(110, '2023-11-16 18:57:56.288', '2023-11-16 18:57:56.288', NULL, 'Kjempefin osteklokke ❤️', '', 1, 1, 'https://www.sagaform.com/global/ditte-ostkupa', 13, 600),
(111, '2023-11-16 22:13:59.644', '2023-11-16 22:14:28.725', NULL, '7 Wonders', 'Brettspill', 1, 10, '', 15, 0),
(112, '2023-11-16 22:14:15.464', '2023-11-16 22:15:38.885', NULL, 'Duffelbag 70L, med bærestropper - ala Anette sik Hagløfs/Helly Hansen etc.', '', 1, 10, '', 15, 0),
(113, '2023-11-18 14:17:25.169', '2023-11-18 14:17:25.169', NULL, 'Asana Relaxed Straight Pant (NinePine)', 'Beste buksa ever, trenger flere sånn at jeg kan være comfy flere dager i uka ✨ Detaljer: Regular inseam, str. M. Svart, navy eller grå (anything goes).', 1, 1, 'https://www.ninepine.no/collections/bestselling-bottoms-1/products/asana-relaxed-straight-zip-pant?variant=40194399502417', 1, 1000),
(114, '2023-11-20 17:51:43.584', '2023-11-20 17:51:43.584', NULL, 'Penger til robotstøvsuger', '', 1, 10, '', 15, 0),
(115, '2023-11-20 18:31:25.743', '2023-11-20 18:31:25.743', NULL, 'Ullundertøy fra Cubus ', '', 1, 9, '', 17, 0),
(116, '2023-11-20 18:31:38.627', '2023-11-20 18:31:38.627', NULL, 'Tynne ullsokker', '', 1, 9, '', 17, 0),
(117, '2023-11-20 18:32:31.161', '2023-11-20 18:32:31.161', NULL, 'Ukekalender på papir', '', 1, 9, '', 17, 0);

-- --------------------------------------------------------

--
-- Table structure for table `wishlists`
--

CREATE TABLE `wishlists` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` longtext COLLATE utf8mb4_bin NOT NULL,
  `description` longtext COLLATE utf8mb4_bin,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `owner_id` bigint(20) NOT NULL,
  `date` datetime(3) NOT NULL,
  `claimable` tinyint(1) NOT NULL DEFAULT '0',
  `expires` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `wishlists`
--

INSERT INTO `wishlists` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `enabled`, `owner_id`, `date`, `claimable`, `expires`) VALUES
(1, '2022-11-20 17:23:51.454', '2023-06-19 16:05:33.221', NULL, 'Helenes (p)ønskeliste ❤️', 'Helene sine ønsker for aaalltid✨', 1, 1, '2023-12-31 01:00:00.000', 1, 1),
(2, '2022-11-20 23:41:07.819', '2022-11-25 20:48:27.732', NULL, 'Øystein\'s Ønsker (ØØ)', 'Kun til meg ', 0, 2, '2022-12-31 00:00:00.000', 1, 1),
(3, '2022-11-21 09:24:04.996', '2023-03-25 12:51:31.276', NULL, 'Øystein & Helenes Juleønsker', 'Julegaver til oss ;)', 0, 2, '2022-12-31 00:00:00.000', 1, 1),
(4, '2022-11-21 09:34:46.999', '2022-11-21 09:34:46.999', NULL, 'Ellen & Andreas ', 'Ønske pønske', 1, 3, '2022-12-24 00:00:00.000', 1, 1),
(5, '2022-11-22 10:05:48.081', '2023-11-10 09:41:10.215', NULL, 'Anette 🎄🎁', 'Anette sine ønsker som hun ikke har råd til å kjøpe til seg selv lenger😢', 1, 6, '2025-07-30 02:00:00.000', 1, 0),
(6, '2022-11-23 20:25:56.955', '2022-11-23 20:25:56.955', NULL, 'Beate og Didrik', 'Beate og Didrik', 1, 4, '2022-12-24 00:00:00.000', 1, 1),
(7, '2022-11-26 21:05:22.179', '2023-03-25 12:51:26.694', NULL, 'Øystein\'s Juleønsker', 'yoooo', 0, 2, '2022-12-31 01:00:00.000', 1, 1),
(8, '2022-12-03 17:51:30.325', '2022-12-03 17:51:30.325', NULL, 'Juleønsker', 'Julegaver 2022', 1, 9, '2022-12-24 01:00:00.000', 1, 1),
(9, '2022-12-08 17:26:44.193', '2022-12-08 17:26:44.193', NULL, 'Aurora’s drit kule ønsker', 'beste lnskelisten for jula 2022', 1, 5, '2022-12-31 01:00:00.000', 1, 1),
(10, '2023-02-06 11:15:34.417', '2023-11-05 18:13:42.742', NULL, 'Øysteins Kule Ønsker 🎁', 'Ting jeg trenger og ønsker meg. ', 1, 2, '2024-08-24 02:00:00.000', 1, 0),
(11, '2023-05-22 23:23:31.325', '2023-05-22 23:23:31.325', NULL, 'bursdag aurora ', 'å', 1, 5, '2023-07-04 02:00:00.000', 1, 1),
(12, '2023-05-22 23:24:44.112', '2023-05-22 23:24:44.112', NULL, 'aurora bursdag', 'å', 1, 5, '2023-06-30 02:00:00.000', 1, 1),
(13, '2023-11-01 15:21:37.248', '2023-11-01 17:22:23.682', NULL, 'Øystein og Helenes Elleville Julegaveønsker 🎄', 'Hvis du ønsker å gi noen fine julegaver som både Øystein og Helene vil sette pris på, så har du kommet til riktig sted.', 1, 2, '2023-12-25 01:00:00.000', 1, 1),
(14, '2023-11-15 10:06:01.190', '2023-11-15 10:06:01.190', NULL, 'Kristine-has-finally-fixed-her-password-Xmas 2023 list (T og K liste) ', 'Jul 2023 ønsker K og T ', 1, 7, '2023-12-24 01:00:00.000', 1, 1),
(15, '2023-11-16 22:11:45.836', '2023-11-16 22:11:45.836', NULL, 'Noe kult', 'Tung Anette mener Eirik trenger og ønsker seg selv', 1, 10, '2023-11-16 22:11:45.834', 1, 0),
(16, '2023-11-18 14:09:26.554', '2023-11-20 18:33:18.006', NULL, 'Karianne en gang i året ønsker', 'Får jo bare gaver en gang i året .....', 0, 9, '2023-11-18 14:09:26.553', 1, 0),
(17, '2023-11-20 18:30:54.261', '2023-11-20 18:30:54.261', NULL, 'Karianne jul og bursdag faktsk', 'Jul 2023', 1, 9, '2023-12-25 01:00:00.000', 1, 1);

-- --------------------------------------------------------

--
-- Table structure for table `wishlist_collaborators`
--

CREATE TABLE `wishlist_collaborators` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user` bigint(20) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `wishlist` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `wishlist_collaborators`
--

INSERT INTO `wishlist_collaborators` (`id`, `created_at`, `updated_at`, `deleted_at`, `user`, `enabled`, `wishlist`) VALUES
(1, '2023-11-01 15:21:44.511', '2023-11-01 15:21:44.511', NULL, 1, 1, 13);

-- --------------------------------------------------------

--
-- Table structure for table `wishlist_memberships`
--

CREATE TABLE `wishlist_memberships` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `group` bigint(20) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `wishlist` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `wishlist_memberships`
--

INSERT INTO `wishlist_memberships` (`id`, `created_at`, `updated_at`, `deleted_at`, `group`, `enabled`, `wishlist`) VALUES
(1, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 1, 1, 3),
(2, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 1, 1, 5),
(3, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 1, 1, 1),
(4, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 2, 1, 4),
(5, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 2, 1, 6),
(6, '2022-11-25 00:00:00.000', '2022-11-25 00:00:00.000', NULL, 2, 1, 3),
(7, '2022-11-26 21:08:49.051', '2023-02-06 11:50:59.658', NULL, 2, 0, 7),
(8, '2022-11-26 21:08:49.055', '2023-02-06 11:50:55.850', NULL, 1, 0, 7),
(9, '2022-12-03 17:51:49.815', '2022-12-03 17:51:49.815', NULL, 1, 1, 8),
(10, '2022-12-08 17:26:44.196', '2022-12-08 17:26:44.196', NULL, 1, 1, 9),
(11, '2023-02-06 11:51:09.970', '2023-02-06 11:51:09.970', NULL, 2, 1, 10),
(12, '2023-05-13 19:05:42.195', '2023-05-13 19:05:42.195', NULL, 1, 1, 10),
(13, '2023-05-22 23:24:44.115', '2023-05-22 23:24:44.115', NULL, 1, 1, 12),
(14, '2023-11-01 15:55:21.877', '2023-11-01 15:55:21.877', NULL, 2, 1, 13),
(15, '2023-11-01 15:55:21.882', '2023-11-01 15:55:21.882', NULL, 1, 1, 13),
(16, '2023-11-15 10:06:01.193', '2023-11-15 10:06:01.193', NULL, 1, 1, 14),
(17, '2023-11-16 22:12:20.162', '2023-11-16 22:12:20.162', NULL, 1, 1, 15),
(18, '2023-11-20 18:30:54.266', '2023-11-20 18:30:54.266', NULL, 1, 1, 17);

-- --------------------------------------------------------

--
-- Table structure for table `wish_claims`
--

CREATE TABLE `wish_claims` (
  `id` bigint(20) UNSIGNED NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `wish` bigint(20) NOT NULL,
  `user` bigint(20) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

--
-- Dumping data for table `wish_claims`
--

INSERT INTO `wish_claims` (`id`, `created_at`, `updated_at`, `deleted_at`, `wish`, `user`, `enabled`) VALUES
(1, '2022-12-04 18:25:29.525', '2022-12-04 18:25:29.525', NULL, 20, 2, 1),
(2, '2022-12-04 18:26:37.902', '2022-12-04 18:26:37.902', NULL, 25, 2, 1),
(3, '2022-12-04 18:26:39.435', '2022-12-04 18:26:39.435', NULL, 24, 2, 1),
(4, '2022-12-04 18:26:41.248', '2022-12-04 18:26:41.248', NULL, 23, 2, 1),
(5, '2022-12-04 18:50:32.855', '2022-12-04 18:52:25.328', NULL, 36, 2, 0),
(6, '2022-12-05 12:07:59.450', '2022-12-05 12:07:59.450', NULL, 7, 4, 1),
(7, '2022-12-05 12:08:03.730', '2022-12-05 12:08:03.730', NULL, 8, 4, 1),
(8, '2022-12-06 17:44:01.497', '2022-12-06 17:44:01.497', NULL, 36, 6, 1),
(9, '2022-12-07 14:10:04.179', '2022-12-07 14:10:04.179', NULL, 5, 3, 1),
(10, '2022-12-07 14:10:06.136', '2022-12-07 14:10:06.136', NULL, 6, 3, 1),
(11, '2022-12-08 17:14:39.469', '2022-12-08 17:14:39.469', NULL, 28, 9, 1),
(12, '2022-12-22 10:43:03.667', '2022-12-22 10:43:03.667', NULL, 26, 6, 1),
(13, '2022-12-22 10:43:59.345', '2022-12-22 10:43:59.345', NULL, 52, 6, 1),
(14, '2023-08-16 10:47:13.109', '2023-08-16 10:47:13.109', NULL, 82, 6, 1),
(15, '2023-08-29 08:28:31.665', '2023-08-29 08:28:31.665', NULL, 44, 2, 1),
(16, '2023-11-15 17:57:37.741', '2023-11-15 17:57:37.741', NULL, 63, 1, 1),
(17, '2023-11-20 12:13:03.620', '2023-11-20 12:13:03.620', NULL, 87, 6, 1),
(18, '2023-11-20 18:14:20.883', '2023-11-20 18:14:20.883', NULL, 112, 9, 1),
(19, '2023-11-20 18:29:12.607', '2023-11-20 18:29:12.607', NULL, 95, 9, 1);

--
-- Indexes for dumped tables
--

--
-- Indexes for table `exercises`
--
ALTER TABLE `exercises`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_exercises_deleted_at` (`deleted_at`);

--
-- Indexes for table `goals`
--
ALTER TABLE `goals`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_goals_deleted_at` (`deleted_at`);

--
-- Indexes for table `groups`
--
ALTER TABLE `groups`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_groups_deleted_at` (`deleted_at`);

--
-- Indexes for table `group_memberships`
--
ALTER TABLE `group_memberships`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_group_memberships_deleted_at` (`deleted_at`);

--
-- Indexes for table `invites`
--
ALTER TABLE `invites`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `invite_code` (`invite_code`),
  ADD KEY `idx_invites_deleted_at` (`deleted_at`);

--
-- Indexes for table `news`
--
ALTER TABLE `news`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_news_deleted_at` (`deleted_at`);

--
-- Indexes for table `seasons`
--
ALTER TABLE `seasons`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_seasons_deleted_at` (`deleted_at`);

--
-- Indexes for table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `email` (`email`),
  ADD KEY `idx_users_deleted_at` (`deleted_at`);

--
-- Indexes for table `wishes`
--
ALTER TABLE `wishes`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_wishes_deleted_at` (`deleted_at`);

--
-- Indexes for table `wishlists`
--
ALTER TABLE `wishlists`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_wishlists_deleted_at` (`deleted_at`);

--
-- Indexes for table `wishlist_collaborators`
--
ALTER TABLE `wishlist_collaborators`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_wishlist_collaborators_deleted_at` (`deleted_at`);

--
-- Indexes for table `wishlist_memberships`
--
ALTER TABLE `wishlist_memberships`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_wishlist_memberships_deleted_at` (`deleted_at`);

--
-- Indexes for table `wish_claims`
--
ALTER TABLE `wish_claims`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_wish_claims_deleted_at` (`deleted_at`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `exercises`
--
ALTER TABLE `exercises`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `goals`
--
ALTER TABLE `goals`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `groups`
--
ALTER TABLE `groups`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=4;

--
-- AUTO_INCREMENT for table `group_memberships`
--
ALTER TABLE `group_memberships`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=16;

--
-- AUTO_INCREMENT for table `invites`
--
ALTER TABLE `invites`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=15;

--
-- AUTO_INCREMENT for table `news`
--
ALTER TABLE `news`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=2;

--
-- AUTO_INCREMENT for table `seasons`
--
ALTER TABLE `seasons`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `users`
--
ALTER TABLE `users`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=11;

--
-- AUTO_INCREMENT for table `wishes`
--
ALTER TABLE `wishes`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=118;

--
-- AUTO_INCREMENT for table `wishlists`
--
ALTER TABLE `wishlists`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=18;

--
-- AUTO_INCREMENT for table `wishlist_collaborators`
--
ALTER TABLE `wishlist_collaborators`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=2;

--
-- AUTO_INCREMENT for table `wishlist_memberships`
--
ALTER TABLE `wishlist_memberships`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=19;

--
-- AUTO_INCREMENT for table `wish_claims`
--
ALTER TABLE `wish_claims`
  MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=20;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
