from flask import Flask, request, jsonify
import os

app = Flask(__name__)


@app.route("/diagnostics")
def diagnostics():
    target = request.args.get("target", "localhost")
    count = request.args.get("count", "1")
    command = "ping -c " + count + " " + target
    output = os.system(command)
    return jsonify({"exit_code": output})
