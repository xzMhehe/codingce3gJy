SELECT COUNT(*) AS with_img FROM hxxy_map_npcs WHERE img<>'' AND img IS NOT NULL; SELECT COUNT(*) AS no_img FROM hxxy_map_npcs WHERE img='' OR img IS NULL;
