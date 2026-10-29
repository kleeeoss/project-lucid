const express = require("express");
const path = require("path");
const fs = require("fs");
const { exec } = require("child_process");
const db = require("./db");

const app = express();
app.use(express.json());

// Insecure Secret
const awsAccessKey = "AKIA1234567890ABCDEF";

// SQL Injection via query param
app.get("/users", (req, res) => {
  const id = req.query.id;
  const q = "SELECT * FROM users WHERE id = " + id;
  db.query(q);
});

// Command Injection via body param
app.post("/ping", (req, res) => {
  const host = req.body.host;
  exec("ping -c 1 " + host);
});

// Path Traversal via path.join
app.get("/file", (req, res) => {
  const file = req.query.file;
  const filePath = path.join("/var/data", file);
  fs.readFile(filePath);
});
