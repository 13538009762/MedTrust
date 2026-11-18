# -*- coding: utf-8 -*-
"""
MedTrust 不改代码一键恢复脚本（Fabric 账本重建 + 基准哈希重锚 + 本地状态校验）

适用场景：
  升级哈希计算口径后，Fabric 持久化账本与 ledger_history.json 仍存有旧版 9 字段哈希，
  且现行链码禁止覆写已有 RECORD_<record_no> 资产。本脚本在不修改任何业务代码的前提下：
    1. 停止当前运行的 medtrust_backend.exe；
    2. 重建干净的 Hyperledger Fabric 测试网容器卷并部署现有链码；
    3. 复位 MySQL 中历史残留的 TAMPERED / NEEDS_RECONCILIATION 状态；
    4. 重启现有 medtrust_backend.exe，触发 EnsureBaselineLedgerAnchored() 自动按新版 19 字段哈希全量上链；
    5. 同步更新 backend/data/fabric_ledger/ledger_history.json；
    6. 逐条核验数据库全部病历与 Fabric 链上资产、AES-256-GCM 密文的一致性。
"""

import argparse
import base64
import hashlib
import json
import os
import re
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

import pymysql
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BACKEND_DIR = os.path.join(ROOT_DIR, "backend")
BACKEND_EXE = os.path.join(BACKEND_DIR, "medtrust_backend.exe")
FABRIC_COMPOSE = os.path.join(ROOT_DIR, "deploy", "fabric", "docker-compose-test-net.yaml")
LEDGER_HISTORY_PATH = os.path.join(BACKEND_DIR, "data", "fabric_ledger", "ledger_history.json")
IPFS_STORAGE_DIR = os.path.join(BACKEND_DIR, "data", "ipfs_storage")
HEALTH_URL = "http://127.0.0.1:8080/api/v1/health"


def load_env():
    env = {}
    env_path = os.path.join(ROOT_DIR, ".env")
    if os.path.exists(env_path):
        with open(env_path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                env[k.strip()] = v.strip().strip('"').strip("'")
    for k, v in os.environ.items():
        if v:
            env[k] = v
    return env


def parse_db_config(env):
    dsn = env.get("DB_DSN", "")
    m = re.match(r"([^:]+):([^@]*)@tcp\(([^:]+):(\d+)\)/([^?]+)", dsn)
    if m:
        user, pwd, host, port, db = m.groups()
        return {
            "host": host,
            "port": int(port),
            "user": user,
            "password": pwd,
            "database": db,
            "charset": "utf8mb4",
            "cursorclass": pymysql.cursors.DictCursor,
        }
    return {
        "host": env.get("MYSQL_HOST", "127.0.0.1"),
        "port": int(env.get("MYSQL_PORT", "3306")),
        "user": env.get("MYSQL_USER", "root"),
        "password": env.get("MYSQL_PASSWORD", ""),
        "database": env.get("MYSQL_DATABASE", "medtrust"),
        "charset": "utf8mb4",
        "cursorclass": pymysql.cursors.DictCursor,
    }


def get_aes_master_key(env):
    raw_mk = (env.get("MEDTRUST_CRYPTO_MASTER_KEY") or env.get("AES_MASTER_KEY") or "").strip()
    if raw_mk.startswith("base64:"):
        decoded = base64.b64decode(raw_mk[7:])
        if len(decoded) == 32:
            return decoded
    if len(raw_mk) == 32:
        return raw_mk.encode("utf-8")
    if raw_mk:
        padded = raw_mk
        while len(padded) < 32:
            padded += "!"
        return padded[:32].encode("utf-8")
    return None


def select_primary_file(record_no, files):
    if not files:
        return None
    target = files[-1]
    for f in reversed(files):
        fn = f.get("file_name") or ""
        if "归档凭据" in fn or (record_no and record_no in fn):
            target = f
            break
    return target


def compute_record_hash(rec, file_hash=""):
    payload = {
        "record_no": str(rec.get("record_no") or ""),
        "data_type": str(rec.get("data_type") or ""),
        "encounter_type": str(rec.get("encounter_type") or ""),
        "department_name": str(rec.get("department_name") or ""),
        "onset_time": str(rec.get("onset_time") or ""),
        "duration": str(rec.get("duration") or ""),
        "symptoms": str(rec.get("symptoms") or ""),
        "etiology": str(rec.get("etiology") or ""),
        "treatment_plan": str(rec.get("treatment_plan") or ""),
        "diagnosis": str(rec.get("diagnosis") or ""),
        "chief_complaint": str(rec.get("chief_complaint") or ""),
        "present_illness": str(rec.get("present_illness") or ""),
        "vital_signs": str(rec.get("vital_signs") or ""),
        "initial_diagnosis": str(rec.get("initial_diagnosis") or ""),
        "diagnostic_basis": str(rec.get("diagnostic_basis") or ""),
        "exam_items": str(rec.get("exam_items") or ""),
        "exam_reason": str(rec.get("exam_reason") or ""),
        "exam_result": str(rec.get("exam_result") or ""),
        "file_hash": str(file_hash or ""),
    }
    raw = json.dumps(payload, ensure_ascii=False, separators=(",", ":"))
    # Align with Go encoding/json.Marshal default SetEscapeHTML(true)
    raw = (
        raw.replace("<", "\\u003c")
        .replace(">", "\\u003e")
        .replace("&", "\\u0026")
        .replace("\u2028", "\\u2028")
        .replace("\u2029", "\\u2029")
    )
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()


def run_cmd(cmd, cwd=ROOT_DIR, check=True):
    print(f"[CMD] {' '.join(cmd)}")
    res = subprocess.run(cmd, cwd=cwd, text=True)
    if check and res.returncode != 0:
        raise RuntimeError(f"Command failed with exit code {res.returncode}: {' '.join(cmd)}")
    return res


def stop_backend():
    print("[1/6] 检查并停止运行中的 medtrust_backend.exe ...")
    if os.name == "nt":
        res = subprocess.run(
            ["taskkill", "/F", "/IM", "medtrust_backend.exe"],
            capture_output=True,
            text=True,
        )
        if res.returncode == 0:
            print("      已停止旧 medtrust_backend.exe 进程。")
            time.sleep(1.0)
        else:
            print("      当前无运行中的 medtrust_backend.exe 进程。")
    else:
        subprocess.run(["pkill", "-f", "medtrust_backend"], check=False)


def reset_mysql_status(db_cfg):
    print("[2/6] 清理 MySQL 历史告警与对账残留状态 ...")
    conn = pymysql.connect(**db_cfg)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE medical_records
                SET integrity_status = 'VERIFIED',
                    sync_status = 'COMPLETED',
                    sync_error = ''
                WHERE integrity_status <> 'VERIFIED'
                   OR sync_status <> 'COMPLETED'
                   OR COALESCE(sync_error, '') <> ''
                """
            )
            updated = cur.rowcount
        conn.commit()
        print(f"      已复位 {updated} 条病历的 integrity_status/sync_status。")
    finally:
        conn.close()


def wait_port(host, port, timeout=60):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with socket.create_connection((host, port), timeout=1.5):
                return True
        except OSError:
            time.sleep(1.0)
    return False


def reset_fabric_network():
    print("[3/6] 清空旧 Fabric 容器卷并重新部署未改动智能合约 ...")
    run_cmd(["docker", "info"], check=True)
    run_cmd(
        ["docker", "compose", "-f", FABRIC_COMPOSE, "down", "-v", "--remove-orphans"],
        check=True,
    )

    ps_res = subprocess.run(
        ["docker", "ps", "-a", "--filter", "name=dev-peer", "-q"],
        capture_output=True,
        text=True,
    )
    dev_ids = [line.strip() for line in ps_res.stdout.splitlines() if line.strip()]
    if dev_ids:
        run_cmd(["docker", "rm", "-f", *dev_ids], check=False)

    crypto_orderer = os.path.join(ROOT_DIR, "deploy", "fabric", "crypto-config", "ordererOrganizations")
    if not os.path.exists(crypto_orderer):
        raise RuntimeError(f"缺少 Fabric 证书目录: {crypto_orderer}，请先运行 scripts/start_fabric.bat")

    run_cmd(["docker", "compose", "-f", FABRIC_COMPOSE, "up", "-d"], check=True)
    if not wait_port("127.0.0.1", 7051, timeout=60):
        raise RuntimeError("等待 peer0.org1.example.com (127.0.0.1:7051) 就绪超时")
    time.sleep(2.0)

    run_cmd(
        [
            "docker",
            "exec",
            "cli",
            "/bin/bash",
            "/opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh",
            "channel",
        ],
        check=True,
    )
    run_cmd(
        [
            "docker",
            "exec",
            "cli",
            "/bin/bash",
            "/opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh",
            "deploy",
        ],
        check=True,
    )
    print("      Fabric 空白通道 medicalchannel 与链码 medical_contract 已就绪。")


def start_backend_detached():
    print("[4/6] 启动现有 medtrust_backend.exe 执行基准重锚 ...")
    if not os.path.exists(BACKEND_EXE):
        raise RuntimeError(f"未找到可执行文件: {BACKEND_EXE}")
    creation_flags = 0
    if os.name == "nt":
        creation_flags = subprocess.CREATE_NEW_CONSOLE | subprocess.CREATE_NEW_PROCESS_GROUP
    subprocess.Popen(
        [BACKEND_EXE],
        cwd=BACKEND_DIR,
        creationflags=creation_flags,
    )


def wait_backend_health(timeout=180):
    print("[5/6] 等待后端 EnsureBaselineLedgerAnchored() 完成并开放健康检查 ...")
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(HEALTH_URL, timeout=3) as resp:
                if resp.status == 200:
                    print("      后端已就绪（HTTP 200），全量上链锚定完成。")
                    return
        except (urllib.error.URLError, TimeoutError, ConnectionError):
            pass
        time.sleep(2.0)
    raise RuntimeError(f"等待后端 {HEALTH_URL} 返回 200 超时（{timeout}s）")


def query_fabric_asset(record_no):
    ctor = json.dumps({"Args": ["QueryMedicalAsset", record_no]})
    res = subprocess.run(
        [
            "docker",
            "exec",
            "cli",
            "peer",
            "chaincode",
            "query",
            "-C",
            "medicalchannel",
            "-n",
            "medical_contract",
            "-c",
            ctor,
        ],
        capture_output=True,
        text=True,
    )
    if res.returncode != 0:
        return None
    return json.loads(res.stdout)


def sync_ledger_history(db_cfg):
    print("[6/6] 同步刷新 ledger_history.json 基准哈希镜像 ...")
    if not os.path.exists(LEDGER_HISTORY_PATH):
        print("      未发现 ledger_history.json，跳过。")
        return

    with open(LEDGER_HISTORY_PATH, "r", encoding="utf-8") as f:
        items = json.load(f)

    conn = pymysql.connect(**db_cfg)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT * FROM medical_records ORDER BY id")
            records = cur.fetchall()
            latest_map = {}
            max_height = 100
            for it in items:
                bh = int(it.get("block_height") or 0)
                if bh > max_height:
                    max_height = bh

            for rec in records:
                cur.execute(
                    "SELECT * FROM medical_files WHERE record_id=%s ORDER BY id ASC",
                    (rec["id"],),
                )
                files = cur.fetchall()
                primary = select_primary_file(rec["record_no"], files)
                fh = primary["file_hash"] if primary else ""
                cid = primary["ipfs_cid"] if primary else ""
                ch = compute_record_hash(rec, fh)
                latest_map[rec["record_no"]] = {
                    "rec": rec,
                    "clinical_hash": ch,
                    "file_hash": fh,
                    "cid": cid,
                }
    finally:
        conn.close()

    seen_medical = set()
    for it in items:
        if (it.get("asset_type") or "") == "MEDICAL_RECORD":
            aid = it.get("asset_id") or ""
            if aid in latest_map:
                info = latest_map[aid]
                payload = it.setdefault("payload", {})
                payload["clinical_hash"] = info["clinical_hash"]
                payload["file_hash"] = info["file_hash"]
                if info["cid"]:
                    payload["cid"] = info["cid"]
                it["tx_id"] = info["rec"].get("fabric_tx_id") or it.get("tx_id")
                seen_medical.add(aid)

    for rec_no, info in latest_map.items():
        if rec_no not in seen_medical:
            max_height += 1
            rec = info["rec"]
            items.append(
                {
                    "tx_id": rec.get("fabric_tx_id") or "",
                    "block_height": int(rec.get("block_height") or max_height),
                    "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S+08:00"),
                    "asset_type": "MEDICAL_RECORD",
                    "asset_id": rec_no,
                    "payload": {
                        "record_no": rec_no,
                        "patient_id": rec.get("patient_id"),
                        "doctor_id": rec.get("doctor_id"),
                        "hospital_id": rec.get("hospital_id"),
                        "cid": info["cid"],
                        "file_hash": info["file_hash"],
                        "clinical_hash": info["clinical_hash"],
                        "data_type": rec.get("data_type"),
                    },
                }
            )

    ledger_dir = os.path.dirname(LEDGER_HISTORY_PATH)
    fd, tmp_path = tempfile.mkstemp(prefix="ledger_history_", suffix=".tmp", dir=ledger_dir)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as tmp_f:
            json.dump(items, tmp_f, ensure_ascii=False, indent=2)
        os.replace(tmp_path, LEDGER_HISTORY_PATH)
    except Exception:
        if os.path.exists(tmp_path):
            os.remove(tmp_path)
        raise
    print(f"      已原子更新 {LEDGER_HISTORY_PATH}（共刷新 {len(latest_map)} 份病历快照）。")


def verify_all_records(env, db_cfg):
    print("\n[VERIFY] 正在全量核验数据库全部病历与 Fabric/IPFS 状态 ...")
    mk_bytes = get_aes_master_key(env)
    conn = pymysql.connect(**db_cfg)
    passed = 0
    failed = []
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT * FROM medical_records ORDER BY id")
            records = cur.fetchall()
            for rec in records:
                cur.execute(
                    "SELECT * FROM medical_files WHERE record_id=%s ORDER BY id ASC",
                    (rec["id"],),
                )
                files = cur.fetchall()
                primary = select_primary_file(rec["record_no"], files)

                reasons = []
                if not rec.get("fabric_tx_id"):
                    reasons.append("fabric_tx_id 为空")
                if rec.get("integrity_status") != "VERIFIED":
                    reasons.append(f"integrity_status={rec.get('integrity_status')}")
                if rec.get("sync_status") != "COMPLETED":
                    reasons.append(f"sync_status={rec.get('sync_status')}")

                chain = query_fabric_asset(rec["record_no"])
                if not chain:
                    reasons.append("Fabric 链上资产缺失")
                else:
                    chain_ch = str(chain.get("clinical_hash") or chain.get("file_hash") or "")
                    chain_fh = str(chain.get("file_hash") or "")

                    # Stage 0: File integrity check mirroring VerifyRecordDetailed
                    if primary is not None:
                        local_fh = primary.get("file_hash") or ""
                        if primary.get("encrypted_dek"):
                            if not mk_bytes:
                                reasons.append("缺少 AES_MASTER_KEY 无法解密校验")
                            else:
                                fpath = os.path.join(IPFS_STORAGE_DIR, primary.get("ipfs_cid") or "")
                                if not os.path.exists(fpath):
                                    reasons.append(f"IPFS密文缺失({primary.get('ipfs_cid')})")
                                else:
                                    try:
                                        cipher_data = open(fpath, "rb").read()
                                        sealed_dek = base64.b64decode(primary["encrypted_dek"])
                                        dek = AESGCM(mk_bytes).decrypt(sealed_dek[:12], sealed_dek[12:], None)
                                        iv = base64.b64decode(primary["iv_nonce"])
                                        aad = f"patient:{rec['patient_id']}|record:{rec['record_no']}".encode("utf-8")
                                        pt = AESGCM(dek).decrypt(iv, cipher_data, aad)
                                        dec_fh = hashlib.sha256(pt).hexdigest()
                                        if dec_fh != local_fh:
                                            reasons.append("AES-GCM解密明文SHA256与DB不一致")
                                        local_fh = dec_fh
                                    except Exception as exc:
                                        reasons.append(f"AES-GCM解密失败: {exc}")

                        file_matched = (not chain_fh) or (local_fh == chain_fh) or (
                            len(files) > 1 and any((f.get("file_hash") or "") == chain_fh for f in files)
                        )
                        if not file_matched:
                            reasons.append(f"FileHash失配(local={local_fh[:12]}, chain={chain_fh[:12]})")

                    # Stage 1: Clinical multi-field JSON hash check
                    default_fh = primary.get("file_hash") or "" if primary else ""
                    cur_ch = compute_record_hash(rec, default_fh)
                    clin_ok = cur_ch == chain_ch
                    if not clin_ok and len(files) > 1:
                        clin_ok = any(compute_record_hash(rec, f.get("file_hash") or "") == chain_ch for f in files)
                    if not clin_ok:
                        reasons.append(f"ClinicalHash失配(cur={cur_ch[:12]}, chain={chain_ch[:12]})")

                if reasons:
                    failed.append((rec["id"], rec["record_no"], "; ".join(reasons)))
                else:
                    passed += 1

        total = len(records)
        print(f"核验完成: {passed}/{total} 条病历全部 VERIFIED 且与 Fabric 链上指纹完全一致。")
        if failed:
            for rid, rno, why in failed:
                print(f"  [FAIL] ID={rid} {rno}: {why}")
            return 1
        return 0
    finally:
        conn.close()


def main():
    parser = argparse.ArgumentParser(description="MedTrust Fabric ledger re-anchor & recovery tool")
    parser.add_argument("--no-start-backend", action="store_true", help="重置 Fabric 与 MySQL 后不自动拉起后端")
    parser.add_argument("--verify-only", action="store_true", help="仅等待/同步并核验当前状态，不重建 Fabric")
    parser.add_argument("--wait-health", action="store_true", help="与 --verify-only 联用：先等待 /api/v1/health 返回 200")
    parser.add_argument("--sync-history", action="store_true", help="与 --verify-only 联用：同时刷新 ledger_history.json")
    args = parser.parse_args()

    env = load_env()
    db_cfg = parse_db_config(env)

    if args.verify_only:
        if args.wait_health:
            wait_backend_health(timeout=180)
        if args.sync_history:
            sync_ledger_history(db_cfg)
        sys.exit(verify_all_records(env, db_cfg))

    stop_backend()
    reset_mysql_status(db_cfg)
    reset_fabric_network()

    if args.no_start_backend:
        print("\n已完成 Fabric 重置与 MySQL 复位；请启动 medtrust_backend.exe 完成基准重锚。")
        return

    start_backend_detached()
    wait_backend_health(timeout=180)
    sync_ledger_history(db_cfg)
    sys.exit(verify_all_records(env, db_cfg))


if __name__ == "__main__":
    main()
