// src/app/api/scan/route.ts
import { NextRequest, NextResponse } from "next/server";
import { query } from "@/lib/db";
import { execFile } from "child_process";
import { promisify } from "util";
import fs from "fs";
import path from "path";
import os from "os";
import crypto from "crypto";

const execFileAsync = promisify(execFile);

export async function POST(req: NextRequest) {
    const startTime = Date.now();
    let tempDir: string | null = null;
    let scanId: string | null = null;

    try {
        const body = await req.json();
        const { repo_url, branch } = body;

        if (!repo_url || typeof repo_url !== "string") {
            return NextResponse.json(
                { success: false, error: "Missing required parameter: repo_url" },
                { status: 400 }
            );
        }

        const trimmedUrl = repo_url.trim();
        const githubRegex = /^https:\/\/github\.com\/([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+?)(\.git)?\/?$/;
        const match = trimmedUrl.match(githubRegex);

        if (!match) {
            return NextResponse.json(
                {
                    success: false,
                    error: "Invalid GitHub URL. Must be in the format: https://github.com/owner/repository",
                },
                { status: 400 }
            );
        }

        const owner = match[1];
        const repoName = match[2];
        const fullName = `${owner}/${repoName}`;
        const targetBranch = (branch && typeof branch === "string" && branch.trim()) || "main";

        // Generate deterministic 63-bit integer for repository ID
        const hash = crypto.createHash("sha256").update(fullName).digest("hex");
        const repoId = (BigInt("0x" + hash.slice(0, 15)) & BigInt("0x7fffffffffffffff")).toString();

        // Register repository in database
        await query(
            `INSERT INTO repositories (id, full_name, installation_id, default_branch, created_at)
             VALUES ($1, $2, 0, $3, NOW())
             ON CONFLICT (full_name) DO UPDATE SET default_branch = EXCLUDED.default_branch
             RETURNING id;`,
            [repoId, fullName, targetBranch]
        );

        // Create scan run record
        scanId = crypto.randomUUID();
        const commitSha = "HEAD";
        await query(
            `INSERT INTO scan_runs (id, repository_id, pr_number, commit_sha, status, findings_count, started_at)
             VALUES ($1, $2, 0, $3, 'SCANNING', 0, NOW());`,
            [scanId, repoId, commitSha]
        );

        // Create ephemeral temp directory for shallow clone
        tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "lucid-ondemand-"));

        // Shallow clone target repository
        try {
            await execFileAsync("git", ["clone", "--depth", "1", "-b", targetBranch, trimmedUrl, tempDir], {
                timeout: 30000,
            });
        } catch {
            // Fallback: clone without branch specification (default branch)
            await execFileAsync("git", ["clone", "--depth", "1", trimmedUrl, tempDir], {
                timeout: 30000,
            });
        }

        // Locate Go scanner (check compiled binary first, fallback to go run)
        const projectRoot = path.resolve(process.cwd(), "../..");
        const binaryPath = path.join(projectRoot, "engine", "bin", "lucid-scan");

        let scanStdout = "[]";
        if (fs.existsSync(binaryPath)) {
            const { stdout } = await execFileAsync(binaryPath, ["--json", tempDir], {
                timeout: 30000,
            });
            scanStdout = stdout;
        } else {
            const { stdout } = await execFileAsync(
                "go",
                ["run", "./engine/cmd/scan/main.go", "--json", tempDir],
                {
                    cwd: projectRoot,
                    timeout: 45000,
                }
            );
            scanStdout = stdout;
        }

        let findings: any[] = [];
        try {
            findings = JSON.parse(scanStdout);
        } catch (parseErr) {
            console.error("Failed to parse scanner output:", parseErr, scanStdout);
            findings = [];
        }

        // Persist findings in vulnerabilities table
        for (const f of findings) {
            let patch = `// [LUCID-CI DEFENSIVE PATCH]\n// Rule: ${f.rule_id} (${f.cwe})\n// File: ${f.file_path}:${f.line_start}\n// Review and parameterize inputs to sanitize taint sink.`;
            let explanation = `Static AST intraprocedural taint analysis detected user-controlled input propagating to dangerous execution sink in ${f.file_path}:${f.line_start}.`;

            // Attempt AI Remediation service if available
            try {
                const aiRes = await fetch("http://localhost:8000/remediate", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        scan_id: scanId,
                        vulnerability_id: `VULN-${crypto.randomBytes(4).toString("hex")}`,
                        rule_id: f.rule_id,
                        cwe: f.cwe,
                        language: f.file_path.endsWith(".py") ? "python" : "javascript",
                        vulnerable_code: f.vulnerable_code,
                        surrounding_context: f.vulnerable_code,
                        source_info: "User request parameter",
                        sink_info: f.vulnerable_code,
                    }),
                    signal: AbortSignal.timeout(2000),
                });
                if (aiRes.ok) {
                    const aiData = await aiRes.json();
                    if (aiData.suggested_patch) patch = aiData.suggested_patch;
                    if (aiData.explanation) explanation = aiData.explanation;
                }
            } catch {
                // Graceful fallback to deterministic patch
            }

            await query(
                `INSERT INTO vulnerabilities (
                    id, scan_run_id, rule_id, cwe, severity, confidence_score,
                    file_path, line_start, line_end, vulnerable_code,
                    ai_remediation_patch, ai_explanation, sandbox_verified,
                    ast_graph_json, created_at
                 ) VALUES (
                    gen_random_uuid(), $1, $2, $3, $4, $5,
                    $6, $7, $8, $9,
                    $10, $11, $12,
                    $13, NOW()
                 );`,
                [
                    scanId,
                    f.rule_id,
                    f.cwe,
                    f.severity,
                    f.confidence_score,
                    f.file_path,
                    f.line_start,
                    f.line_end,
                    f.vulnerable_code,
                    patch,
                    explanation,
                    false,
                    JSON.stringify(f.ast_graph || { nodes: [], edges: [] }),
                ]
            );
        }

        // Finalize scan run status
        const durationMs = Date.now() - startTime;
        await query(
            `UPDATE scan_runs
             SET status = 'COMPLETED',
                 findings_count = $1,
                 scan_duration_ms = $2,
                 completed_at = NOW()
             WHERE id = $3;`,
            [findings.length, durationMs, scanId]
        );

        return NextResponse.json({
            success: true,
            scan_id: scanId,
            findings_count: findings.length,
            duration_ms: durationMs,
            redirect_url: `/scans/${scanId}`,
        });
    } catch (err: any) {
        console.error("On-demand scan failed:", err);

        if (scanId) {
            try {
                await query(
                    `UPDATE scan_runs SET status = 'FAILED', completed_at = NOW() WHERE id = $1;`,
                    [scanId]
                );
            } catch {}
        }

        return NextResponse.json(
            {
                success: false,
                error: err?.message || "Failed to scan repository. Ensure it is public and accessible.",
            },
            { status: 500 }
        );
    } finally {
        // Ephemeral directory cleanup
        if (tempDir && fs.existsSync(tempDir)) {
            try {
                fs.rmSync(tempDir, { recursive: true, force: true });
            } catch (cleanupErr) {
                console.warn("Failed to cleanup temp scan directory:", cleanupErr);
            }
        }
    }
}
