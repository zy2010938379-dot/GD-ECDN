-- EDNS ECS city mapping seed data.
-- Source: /home/ubuntu/图片/EDNS_ip.png

CREATE TABLE IF NOT EXISTS edns_city_mapping (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    source_group  VARCHAR(64)  NOT NULL,
    region_id     BIGINT       NOT NULL comment '区域id',
    region_name   VARCHAR(64)  NOT NULL comment '区域名称',
    ecs_subnet    VARCHAR(64)  NOT NULL,
    country_name  VARCHAR(32)  DEFAULT '中国',
    province_name VARCHAR(32)  DEFAULT '广东',
    city_name     VARCHAR(32)  DEFAULT NULL,
    city_id       BIGINT       DEFAULT NULL,
    provider_name VARCHAR(64)  DEFAULT NULL,
    provider_id   BIGINT       DEFAULT NULL,
    enabled       TINYINT(1)   NOT NULL DEFAULT 1,
    UNIQUE KEY uk_ecs_subnet (ecs_subnet)
);

INSERT INTO edns_city_mapping (source_group,region_id, region_name, ecs_subnet, city_name, provider_name, enabled) VALUES
('省公司',1, '省公司', '61.142.56.194/32', '省公司', '省公司', 1),
('云浮电信',1, '云浮',   '183.57.187.9/32',  '云浮',   '云浮电信', 1),
('阳江电信',1, '阳江',   '183.57.186.9/32',  '阳江',   '阳江电信', 1),
('肇庆电信',1, '肇庆',   '183.57.185.9/32',  '肇庆',   '肇庆电信', 1),
('茂名电信',1, '茂名',   '183.57.184.9/32',  '茂名',   '茂名电信', 1),
('湛江电信',1, '湛江',   '183.57.183.9/32',  '湛江',   '湛江电信', 1),
('梅州电信',1, '梅州',   '183.57.182.25/32', '梅州',   '梅州电信', 1),
('汕头电信',1, '汕头',   '183.57.181.25/32', '汕头',   '汕头电信', 1),
('惠州电信',1, '惠州',   '183.57.180.9/32',  '惠州',   '惠州电信', 1),
('珠海电信',1, '珠海',   '183.57.179.25/32', '珠海',   '珠海电信', 1),
('江门电信',1, '江门',   '183.57.178.9/32',  '江门',   '江门电信', 1),
('中山电信',1, '中山',   '183.57.177.9/32',  '中山',   '中山电信', 1),
('汕尾电信',1, '汕尾',   '183.57.191.9/32',  '汕尾',   '汕尾电信', 1),
('河源电信',1, '河源',   '183.57.190.9/32',  '河源',   '河源电信', 1),
('韶关电信',1, '韶关',   '183.57.189.9/32',  '韶关',   '韶关电信', 1),
('清远电信',1, '清远',   '183.57.188.9/32',  '清远',   '清远电信', 1),
('潮州电信',1, '潮州',   '61.142.49.9/32',   '潮州',   '潮州电信', 1),
('揭阳电信',1, '揭阳',   '61.142.48.1/32',   '揭阳',   '揭阳电信', 1),
('佛山电信',1, '佛山',   '61.142.46.25/32',  '佛山',   '佛山电信', 1),
('东莞电信',1, '东莞',   '61.142.44.25/32',  '东莞',   '东莞电信', 1),
('深圳电信',1, '深圳',   '61.142.42.97/32',  '深圳',   '深圳电信', 1),
('广州电信',1, '广州',   '61.142.56.193/32', '广州',   '广州电信', 1)
ON DUPLICATE KEY UPDATE
    source_group = VALUES(source_group),
    region_name = VALUES(region_name),
    city_name = VALUES(city_name),
    provider_name = VALUES(provider_name),
    enabled = VALUES(enabled);
