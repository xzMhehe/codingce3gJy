SELECT id,dtx,dty,name,`up`,`down`,`left`,`right`,up_jump,down_jump,left_jump,right_jump,`desc` FROM hxxy_map_nodes WHERE name LIKE '%朱雀%'\G
SELECT COUNT(*) AS nodes_total FROM hxxy_map_nodes;
SELECT COUNT(*) AS npcs_total FROM hxxy_map_npcs;
SELECT COUNT(*) AS spawns_total FROM hxxy_spawns;
SELECT dtx,dty,COUNT(*) c FROM hxxy_map_npcs GROUP BY dtx,dty ORDER BY dtx,dty LIMIT 30;
