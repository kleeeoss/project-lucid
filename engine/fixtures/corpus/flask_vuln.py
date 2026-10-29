from flask import Flask, request, jsonify
import os
import sqlite3

app = Flask(__name__)

# Insecure Secret
GITHUB_TOKEN = "ghp_123456789012345678901234567890123456"

@app.route("/search")
def search():
    name = request.args.get("name", "")
    conn = sqlite3.connect("app.db")
    cursor = conn.cursor()
    # SQLi via f-string
    query = f"SELECT * FROM users WHERE name = '{name}'"
    cursor.execute(query)
    return jsonify(cursor.fetchall())

@app.route("/ping")
def ping():
    host = request.args.get("host", "127.0.0.1")
    # Command injection via os.system
    cmd = "ping -c 1 " + host
    os.system(cmd)
    return jsonify({"status": "pinged"})

@app.route("/read")
def read_doc():
    filename = request.args.get("file", "doc.txt")
    # Path traversal via open
    with open("/data/files/" + filename) as f:
        return f.read()
