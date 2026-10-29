const path = require("path");
const fs = require("fs");

// Tagged template sanitizer
function sql(strings, ...values) {
  return { text: strings.join("?"), values };
}

const handleRequest = async (req, res) => {
  const modName = req.query.module;
  // Dynamic import
  const mod = await import(modName);

  const id = req.query.id;
  // Sanitized tagged template literal
  const query = sql`SELECT * FROM items WHERE id = ${id}`;
  
  // Safe path with path.normalize
  const rawPath = req.query.path;
  const safePath = path.normalize(rawPath);
  fs.readFile(safePath);
};
