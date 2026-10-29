#!/usr/bin/env python3
"""
LUCID-CI — Synthetic Adversarial Payload: Network Egress & AWS Metadata Probe.
Part of TASK-INF-501 Sandbox Penetration Testing Suite.

This script simulates an attacker-controlled PR attempting to:
1. Exfiltrate environment secrets or source code to external DNS (8.8.8.8:53).
2. Probe AWS EC2 Instance Metadata Service (169.254.169.254) to steal IAM instance profile credentials.
3. Resolve public domains via DNS.

Under the Lucid-CI sandbox invariant (--net none), all outbound network calls
MUST fail immediately with 'Network is unreachable' or timeout, verifying complete isolation.
"""

import sys
import socket
import urllib.request
import urllib.error
from typing import Dict, Any


def probe_external_dns_tcp(host: str = "8.8.8.8", port: int = 53, timeout: float = 2.0) -> bool:
    """Attempt outbound TCP socket connection to external DNS."""
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(timeout)
    try:
        sock.connect((host, port))
        sock.close()
        return True  # Egress succeeded -> Breach
    except (OSError, socket.timeout):
        return False  # Blocked -> Isolated


def probe_aws_metadata(url: str = "http://169.254.169.254/latest/meta-data/", timeout: float = 2.0) -> bool:
    """Attempt HTTP request to AWS EC2 instance metadata service."""
    req = urllib.request.Request(url, headers={"User-Agent": "Lucid-CI-PenTest"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            if resp.status == 200:
                return True  # Reached metadata -> Breach
    except (urllib.error.URLError, TimeoutError, OSError):
        pass
    return False  # Blocked -> Isolated


def probe_dns_resolution(hostname: str = "google.com") -> bool:
    """Attempt DNS host resolution."""
    try:
        socket.getaddrinfo(hostname, 80)
        return True
    except (socket.gaierror, OSError):
        return False


def verify_network_isolation() -> Dict[str, Any]:
    """
    Executes all network egress probes and returns a summary dict.
    Returns:
        dict: containing probe results and boolean 'is_isolated'.
    """
    dns_egress = probe_external_dns_tcp()
    metadata_egress = probe_aws_metadata()
    dns_resolution = probe_dns_resolution()

    any_breach = dns_egress or metadata_egress or dns_resolution
    return {
        "dns_egress_tcp": dns_egress,
        "aws_metadata_http": metadata_egress,
        "dns_resolution": dns_resolution,
        "is_isolated": not any_breach,
    }


def main() -> int:
    results = verify_network_isolation()
    print("[PEN_TEST] Probing container network boundary...")
    print(f"  - Outbound TCP 8.8.8.8:53: {'BREACHED' if results['dns_egress_tcp'] else 'BLOCKED'}")
    print(f"  - AWS Metadata 169.254.169.254: {'BREACHED' if results['aws_metadata_http'] else 'BLOCKED'}")
    print(f"  - DNS Resolution (google.com): {'BREACHED' if results['dns_resolution'] else 'BLOCKED'}")

    if not results["is_isolated"]:
        print("[SECURITY_BREACH] Outbound network egress detected! Container isolation failed.")
        return 1

    print("[ISOLATION_VERIFIED] All network egress attempts failed as expected under --net none.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
