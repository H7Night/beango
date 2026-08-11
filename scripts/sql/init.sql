INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(1, '招商银行信用卡(2035)', 'Liabilities:CMBCreditCard:2035', 'asset', '2025-04-26 13:22:02', '2025-04-26 21:25:45.317');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(3, '花呗', 'Liabilities:Huabei', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(4, '余额宝', 'Assets:AliPay:Balance', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(5, 'Yu‘E Bao', 'Assets:AliPay:Balance', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(6, '地铁', 'Expenses:Transport', 'expense', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(7, '中运交通', 'Expenses:Transport', 'expense', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(8, '基金管理', 'Income:Fund', 'income', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(9, '光大银行', 'Assets:CEB:1027', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(10, '招商银行(3229)', 'Assets:CMB:3229', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(11, '零钱', 'Assets:WeChat', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(12, '招商银行储蓄卡(3229)', 'Assets:CMB:3229', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(13, '转账备注', 'Income:RedEnvelope', 'income', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(14, '广发银行储蓄卡(1508)', 'Assets:CGB:41508', 'asset', '2025-04-26 13:22:02', '2025-04-26 13:22:09');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(15, '蚂蚁财富', 'Assets:Alipay:Funds', 'asset', '2026-04-08 10:00:00', '2026-04-08 10:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(16, '信用卡还款', 'Liabilities:CMBCreditCard:2035', 'asset', '2026-04-08 10:00:00', '2026-04-08 10:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(17, '花呗主动还款', 'Liabilities:Huabei', 'asset', '2026-04-08 10:00:00', '2026-04-08 10:00:00');

-- 支付方式补充
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(18, '余额', 'Assets:AliPay:Balance', 'asset', '2026-06-11 00:00:00', '2026-06-11 00:00:00');

-- 支付宝交易分类 → Beancount 支出账户（用于未匹配到具体商户时回退）
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(19, '餐饮美食', 'Expenses:Food:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(20, '日用百货', 'Expenses:Shopping:Home', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(21, '交通出行', 'Expenses:Transport', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(22, '家居家装', 'Expenses:Shopping:Home', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(23, '文化休闲', 'Expenses:Entertainment:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(24, '服饰美容', 'Expenses:Shopping:Clothing', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(25, '数码电器', 'Expenses:Shopping:Digital', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(26, '生活服务', 'Expenses:Home:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(27, '运动健身', 'Expenses:Health:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(28, '医疗健康', 'Expenses:Health:Medical', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');

-- 微信交易分类 → Beancount 支出账户
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(29, '餐饮', 'Expenses:Food:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(30, '购物', 'Expenses:Shopping:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(31, '出行', 'Expenses:Transport', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(32, '生活', 'Expenses:Home:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');
INSERT INTO account_maps (id, keyword, account, type, created_at, updated_at) VALUES(33, '娱乐', 'Expenses:Entertainment:Other', 'expense', '2026-06-11 00:00:00', '2026-06-11 00:00:00');



INSERT INTO beango_configs (id, created_at, updated_at, config_key, config_value, note) VALUES(1, '2025-05-17 13:17:02.133', '2025-05-17 13:17:02.133', 'defaultFolder', '0-default', NULL);
INSERT INTO beango_configs (id, created_at, updated_at, config_key, config_value, note) VALUES(2, '2025-05-17 13:40:59.076', '2025-05-17 13:40:59.076', 'securitFolder', '1-securities', NULL);
INSERT INTO beango_configs (id, created_at, updated_at, config_key, config_value, note) VALUES(3, '2025-05-17 13:41:32.201', '2025-05-17 13:41:32.201', 'outputFolder', './test/out', NULL);
