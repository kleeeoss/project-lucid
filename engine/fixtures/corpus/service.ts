// TypeScript Service Module
export const findUser = async (req) => {
  const id = req.query.id;
  const sql = "SELECT * FROM users WHERE id = " + id;
  db.query(sql);
};

export const getStatus = async () => {
  db.query("SELECT * FROM status");
};
