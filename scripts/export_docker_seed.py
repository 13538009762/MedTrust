# -*- coding: utf-8 -*-
"""
导出当前 MySQL medtrust 数据库全部表结构与全量数据至 deploy/mysql/init.sql
供 Docker 容器首次拉起时自动完成数据无损注入。
"""
import os
import pymysql

output_dir = os.path.join(os.path.dirname(__file__), "..", "deploy", "mysql")
os.makedirs(output_dir, exist_ok=True)
output_file = os.path.join(output_dir, "init.sql")

conn = pymysql.connect(
    host='127.0.0.1',
    user='root',
    password='123456',
    database='medtrust',
    charset='utf8mb4'
)

cur = conn.cursor()

lines = []
lines.append("-- MedTrust Automated Full Schema & Seed Data Initialization for Docker")
lines.append("CREATE DATABASE IF NOT EXISTS medtrust DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
lines.append("USE medtrust;\n")
lines.append("SET NAMES utf8mb4;")
lines.append("SET FOREIGN_KEY_CHECKS = 0;\n")

# 获取所有表
cur.execute("SHOW TABLES")
tables = [row[0] for row in cur.fetchall()]

for table in tables:
    lines.append(f"-- ----------------------------")
    lines.append(f"-- Table structure for {table}")
    lines.append(f"-- ----------------------------")
    lines.append(f"DROP TABLE IF EXISTS `{table}`;")
    
    cur.execute(f"SHOW CREATE TABLE `{table}`")
    create_sql = cur.fetchone()[1]
    lines.append(f"{create_sql};\n")
    
    # 导出表数据
    cur.execute(f"SELECT * FROM `{table}`")
    rows = cur.fetchall()
    if rows:
        cur.execute(f"SHOW COLUMNS FROM `{table}`")
        columns = [f"`{col[0]}`" for col in cur.fetchall()]
        cols_str = ", ".join(columns)
        
        lines.append(f"-- Records of {table}")
        for r in rows:
            vals = []
            for item in r:
                if item is None:
                    vals.append("NULL")
                elif isinstance(item, (int, float)):
                    vals.append(str(item))
                elif isinstance(item, bytes):
                    hex_str = item.hex()
                    vals.append(f"UNHEX('{hex_str}')")
                else:
                    # 转义特殊字符
                    s = str(item).replace("\\", "\\\\").replace("'", "\\'").replace("\n", "\\n").replace("\r", "\\r")
                    vals.append(f"'{s}'")
            lines.append(f"INSERT INTO `{table}` ({cols_str}) VALUES ({', '.join(vals)});")
        lines.append("")

lines.append("SET FOREIGN_KEY_CHECKS = 1;")

with open(output_file, "w", encoding="utf-8") as f:
    f.write("\n".join(lines) + "\n")

print(f"Successfully exported {len(tables)} tables to {output_file}")
