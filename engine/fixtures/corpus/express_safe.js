const express = require("express");
const path = require("path");
const fs = require("fs");
const db = require("./db");

const app = express();
app.use(express.json());

// Safe parameterized SQL
app.get("/users", (req, res) => {
  const id = req.query.id;
  db.query("SELECT * FROM users WHERE id = ?", [id]);
});

// Safe sanitized path with path.resolve
app.get("/file", (req, res) => {
  const file = req.query.file;
  const safePath = path.resolve("/var/data", file);
  fs.readFile(safePath);
});

// Constant query
app.get("/all", (req, res) => {
  db.query("SELECT * FROM users");
});
