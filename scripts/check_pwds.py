import pymysql
import bcrypt

conn = pymysql.connect(
    host='127.0.0.1',
    port=3306,
    user='root',
    password='123456',
    database='medtrust',
    charset='utf8mb4'
)

with conn.cursor(pymysql.cursors.DictCursor) as cur:
    cur.execute("""
        SELECT u.id, u.user_no, u.username, u.password_hash, u.real_name, u.role, u.title, u.phone, u.id_card, u.status,
               h.name as hospital_name, d.name as dept_name
        FROM users u
        LEFT JOIN hospitals h ON u.hospital_id = h.id
        LEFT JOIN departments d ON u.department_id = d.id
        ORDER BY 
            CASE u.role
                WHEN 'admin' THEN 1
                WHEN 'supervisor' THEN 2
                WHEN 'doctor' THEN 3
                WHEN 'patient' THEN 4
                ELSE 5
            END,
            u.id ASC
    """)
    rows = cur.fetchall()

print(f"Total users: {len(rows)}")
for r in rows:
    pwd_ok = False
    try:
        pwd_ok = bcrypt.checkpw(b"123456", r['password_hash'].encode('utf-8'))
    except Exception as e:
        pwd_ok = False
    print(f"{r['username']:<15} | pwd_is_123456: {pwd_ok} | role: {r['role']} | real_name: {r['real_name']}")

conn.close()
