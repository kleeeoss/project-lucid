const express = require("express");
const sqlite3 = require("sqlite3");

const app = express();
const db = new sqlite3.Database("lucid-demo.db");

app.get("/users/:id", (req, res) => {
  const userId = req.params.id;
  const tenant = req.query.tenant;
  const query =
    "SELECT * FROM users WHERE id = " + userId + " AND tenant = '" + tenant + "'";

  db.query(query, (err, rows) => {
    if (err) {
      res.status(500).json({ error: "query failed" });
      return;
    }
    res.json(rows);
  });
});

module.exports = app;
