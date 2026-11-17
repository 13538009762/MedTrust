import pymysql

for port in [3306, 3307]:
    try:
        conn = pymysql.connect(
            host='127.0.0.1',
            port=port,
            user='root',
            password='123456',
            database='medtrust',
            charset='utf8mb4'
        )
        print(f"Connected to MySQL on port {port}")
        with conn.cursor(pymysql.cursors.DictCursor) as cur:
            cur.execute("""
                SELECT u.id, u.user_no, u.username, u.real_name, u.role, u.title, u.phone, u.id_card, u.status,
                       h.name as hospital_name, d.name as dept_name
                FROM users u
                LEFT JOIN hospitals h ON u.hospital_id = h.id
                LEFT JOIN departments d ON u.department_id = d.id
                ORDER BY u.role, u.id
            """)
            rows = cur.fetchall()
            print(f"Total {len(rows)} users found:")
            for r in rows:
                print(f"{r['username']:<15} | {r['real_name']:<12} | {r['role']:<10} | {str(r['title']):<10} | {str(r['hospital_name']):<16} | {str(r['dept_name']):<10} | {r['phone']:<12} | {r['status']}")
        conn.close()
        break
    except Exception as e:
        print(f"Port {port} failed: {e}")
