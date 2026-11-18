import hashlib
import json
import unittest

import server


class TestTamperServerSecurity(unittest.TestCase):
    def setUp(self):
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()

    def test_mutating_endpoints_require_session_token(self):
        for path, body in (
            ("/api/attack", None),
            ("/api/rollback", None),
            ("/api/custom_tamper", {"id": 1, "field": "diagnosis", "value": "x"}),
        ):
            resp = self.client.post(path, json=body)
            self.assertEqual(resp.status_code, 403, f"{path} without token must return 403")
            resp_bad = self.client.post(path, json=body, headers={"X-Tamper-Token": "wrong"})
            self.assertEqual(resp_bad.status_code, 403, f"{path} with wrong token must return 403")

    def test_index_renders_xss_escaping_and_security_headers(self):
        resp = self.client.get("/")
        self.assertEqual(resp.status_code, 200)
        self.assertEqual(resp.headers.get("X-Content-Type-Options"), "nosniff")
        self.assertEqual(resp.headers.get("X-Frame-Options"), "SAMEORIGIN")
        html = resp.get_data(as_text=True)
        self.assertIn("function escapeHtml", html)
        self.assertIn("escapeHtml(r.etiology", html)
        self.assertIn("X-Tamper-Token", html)

    def test_compute_hash_matches_go_json_html_escaping(self):
        rec = {
            "record_no": "REC-TEST-01",
            "treatment_plan": "每日<5g & 监测",
        }
        got = server.compute_hash(rec, "fh123")
        expected_raw = (
            '{"record_no":"REC-TEST-01","data_type":"","encounter_type":"","department_name":"",'
            '"onset_time":"","duration":"","symptoms":"","etiology":"","treatment_plan":"每日\\u003c5g \\u0026 监测",'
            '"diagnosis":"","chief_complaint":"","present_illness":"","vital_signs":"",'
            '"initial_diagnosis":"","diagnostic_basis":"","exam_items":"","exam_reason":"","exam_result":"",'
            '"file_hash":"fh123"}'
        )
        expected = hashlib.sha256(expected_raw.encode("utf-8")).hexdigest()
        self.assertEqual(got, expected)


if __name__ == "__main__":
    unittest.main()
