"""Strictly loopback fake Cloud for the shared Terraform lifecycle scenario."""
from __future__ import annotations

import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import threading
from urllib.error import HTTPError
from urllib.request import Request, urlopen


class Fixture:
    def __init__(self):
        self.subscription = None
        self.database = None
        self.tasks = {}
        self.next_task = 1
        self.errors = []
        self.events = []
        self.lock = threading.RLock()
        fixture = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_args):
                pass

            def do_GET(self):
                fixture.handle(self)

            def do_POST(self):
                fixture.handle(self)

            def do_PUT(self):
                fixture.handle(self)

            def do_DELETE(self):
                fixture.handle(self)

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.url = f"http://127.0.0.1:{self.server.server_port}/v1"

    def close(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def reply(self, handler, status, payload):
        data = json.dumps(payload).encode()
        handler.send_response(status)
        handler.send_header("Content-Type", "application/json")
        handler.send_header("Content-Length", str(len(data)))
        handler.end_headers()
        handler.wfile.write(data)

    def task(self, handler, resource=None):
        task_id = f"fixture-{self.next_task}"
        self.next_task += 1
        self.tasks[task_id] = {"status": "processing-completed", "response": {"resourceId": resource}}
        self.reply(handler, 200, {"taskId": task_id})

    def handle(self, handler):
        with self.lock:
            method, path = handler.command, handler.path.split("?", 1)[0]
            if handler.headers.get("X-Api-Key") != "fake" or handler.headers.get("X-Api-Secret-Key") != "fake":
                self.errors.append("invalid auth")
                self.reply(handler, 403, {"error": "fixture credentials required"})
                return
            self.events.append((method, path))
            body = {}
            if method in ("POST", "PUT"):
                try:
                    body = json.loads(handler.rfile.read(int(handler.headers.get("Content-Length", "0"))))
                except (ValueError, TypeError):
                    self.errors.append("invalid body")
                    self.reply(handler, 400, {"error": "invalid body"})
                    return
            sub = "/v1/fixed/subscriptions/101"
            db = sub + "/databases/202"
            if method == "GET" and path == "/v1/fixed/subscriptions":
                self.reply(handler, 200, {"subscriptions": [self.subscription] if self.subscription else []})
            elif method == "POST" and path == "/v1/fixed/subscriptions" and self.subscription is None:
                self.subscription = {"id": 101, "name": body.get("name"), "status": "active", "planId": 1, "paymentMethodType": "credit-card", "creationDate": "2025-01-01T00:00:00Z"}
                self.task(handler, 101)
            elif method == "GET" and path == sub:
                self.reply(handler, 200 if self.subscription else 404, self.subscription or {"error": "not found"})
            elif method == "DELETE" and path == sub and self.subscription is not None and self.database is None:
                self.subscription = None
                self.task(handler)
            elif method == "POST" and path == sub + "/databases" and self.subscription is not None and self.database is None:
                self.database = {"databaseId": 202, "name": body.get("name"), "protocol": "redis", "provider": "GCP", "region": "us-central1", "status": "active", "dataPersistence": "none", "dataEvictionPolicy": body.get("dataEvictionPolicy"), "replication": False, "activatedOn": "2025-01-01T00:00:00Z", "security": {"defaultUserEnabled": True, "sourceIps": ["0.0.0.0/0"]}, "modules": [], "alerts": []}
                self.task(handler, 202)
            elif method == "GET" and path == db:
                self.reply(handler, 200 if self.database else 404, self.database or {"error": "not found"})
            elif method == "PUT" and path == db and self.database is not None:
                self.database["dataEvictionPolicy"] = body.get("dataEvictionPolicy")
                self.task(handler)
            elif method == "DELETE" and path == db and self.database is not None:
                self.database = None
                self.task(handler)
            elif method == "GET" and path == db + "/tags" and self.database is not None:
                self.reply(handler, 200, {"tags": []})
            elif method == "PUT" and path == db + "/tags" and self.database is not None:
                self.reply(handler, 200, {"tags": []})
            elif method == "GET" and path.startswith("/v1/tasks/") and path.rsplit("/", 1)[-1] in self.tasks:
                self.reply(handler, 200, self.tasks[path.rsplit("/", 1)[-1]])
            else:
                self.errors.append(f"unexpected {method} {path}")
                self.reply(handler, 404, {"error": "unexpected fixture request"})

    def read(self, path):
        req = Request(self.url + path, headers={"X-Api-Key": "fake", "X-Api-Secret-Key": "fake"})
        try:
            with urlopen(req, timeout=5) as resp:
                return resp.status, json.load(resp)
        except HTTPError as exc:
            return exc.code, None
