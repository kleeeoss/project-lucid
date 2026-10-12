"use client";

import { useEffect, useState } from "react";

export type MemberRole =
    | "Team Lead"
    | "Security Lead"
    | "Developer"
    | "DevOps Engineer"
    | "Admin"
    | "CI Agent";

export interface TeamMember {
    id: string;
    name: string;
    email: string;
    role: MemberRole;
    isLead: boolean;
    avatar: string;
    github: string;
    lastActive: string;
    color?: string;
}

export const DEFAULT_MEMBERS: TeamMember[] = [
    {
        id: "1",
        name: "DevSecOps Lead",
        email: "lead@lucid-security.org",
        role: "Team Lead",
        isLead: true,
        avatar: "DL",
        github: "@devsec-lead",
        lastActive: "Now",
        color: "from-primary-container to-tertiary-container",
    },
    {
        id: "2",
        name: "Sarah Chen",
        email: "sarah@lucid-security.org",
        role: "Security Lead",
        isLead: false,
        avatar: "SC",
        github: "@sarahchen",
        lastActive: "2h ago",
        color: "from-tertiary to-secondary",
    },
    {
        id: "3",
        name: "Alex Rivera",
        email: "alex@lucid-security.org",
        role: "Developer",
        isLead: false,
        avatar: "AR",
        github: "@alexdoe",
        lastActive: "1d ago",
        color: "from-secondary to-primary",
    },
    {
        id: "4",
        name: "Dev Bot",
        email: "bot@lucid-security.org",
        role: "CI Agent",
        isLead: false,
        avatar: "DB",
        github: "@lucid-bot",
        lastActive: "Always",
        color: "from-primary/50 to-on-surface-variant/30",
    },
];

const STORAGE_KEY = "lucid_ci_team_members";
const EVENT_NAME = "lucid:team-updated";

export function getStoredMembers(): TeamMember[] {
    if (typeof window === "undefined") return DEFAULT_MEMBERS;
    try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (!raw) return DEFAULT_MEMBERS;
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    } catch {
        // fallback
    }
    return DEFAULT_MEMBERS;
}

export function saveStoredMembers(members: TeamMember[]): void {
    if (typeof window === "undefined") return;
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(members));
        window.dispatchEvent(new CustomEvent(EVENT_NAME, { detail: members }));
    } catch (e) {
        console.error("Failed to save team members", e);
    }
}

export function getStoredLead(): TeamMember {
    const members = getStoredMembers();
    return members.find((m) => m.isLead) || members[0] || DEFAULT_MEMBERS[0];
}

export function useTeamLead(): TeamMember {
    const [lead, setLead] = useState<TeamMember>(DEFAULT_MEMBERS[0]);

    useEffect(() => {
        setLead(getStoredLead());

        const handleUpdate = () => {
            setLead(getStoredLead());
        };

        window.addEventListener(EVENT_NAME, handleUpdate);
        window.addEventListener("storage", handleUpdate);
        return () => {
            window.removeEventListener(EVENT_NAME, handleUpdate);
            window.removeEventListener("storage", handleUpdate);
        };
    }, []);

    return lead;
}
