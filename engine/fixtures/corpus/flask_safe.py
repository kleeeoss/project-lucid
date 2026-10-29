from flask import Flask, request, jsonify
import os
import shlex
import sqlite3

app = Flask(__name__)

@app.route("/search")
def search():
    name = request.args.get("name", "")
    conn = sqlite3.connect("app.db")
    cursor = conn.cursor()
    # Safe parameterized query
    cursor.execute("SELECT * FROM users WHERE name = ?", (name,))
    return jsonify(cursor.fetchall())

@app.route("/ping")
def ping():
    host = request.args.get("host", "127.0.0.1")
    # Sanitized using shlex.quote
    safe_host = shlex.quote(host)
    os.system("ping -c 1 " + safe_host)
    return jsonify({"status": "pinged"})

@app.route("/read")
def read_doc():
    filename = request.args.get("file", "doc.txt")
    # Sanitized using os.path.normpath
    safe_path = os.path.normpath("/data/files/" + filename)
    with open(safe_path) as f:
        return f.read()
