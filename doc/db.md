# 授权码表

```shell
CREATE TABLE `authcode` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `code` varchar(64) NOT NULL COMMENT '授权码',
  `type` tinyint NOT NULL COMMENT '有效期类型 对应 ValidityType 枚举',
  `expiry_time` datetime NOT NULL COMMENT '过期时间',
  `is_active` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否激活 0=未激活 1=已激活',
  `activated_at` datetime DEFAULT NULL COMMENT '激活时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`) COMMENT '授权码唯一'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='授权码表';
```