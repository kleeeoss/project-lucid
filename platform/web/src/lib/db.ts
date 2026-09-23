import { Pool } from "pg";

const connectionString =
    process.env.DATABASE_URL || "postgres://lucid_user:lucid_password@localhost:5433/lucid_ci";

// Singleton pool pattern for Next.js hot-reloading
const globalForPg = globalThis as unknown as { pgPool: Pool };

export const pool =
    globalForPg.pgPool ||
    new Pool({
        connectionString,
        max: 10,
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 5000,
    });

if (process.env.NODE_ENV !== "production") {
    globalForPg.pgPool = pool;
}

export async function query(text: string, params?: any[]) {
    const client = await pool.connect();
    try {
        return await client.query(text, params);
    } finally {
        client.release();
    }
}