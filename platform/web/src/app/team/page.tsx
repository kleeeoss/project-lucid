"use client";

import { useEffect, useState } from "react";
import {
    Users,
    Shield,
    Mail,
    Clock,
    Crown,
    UserPlus,
    Trash2,
    Edit2,
    Check,
    X,
    Star,
    Github,
} from "lucide-react";
import {
    TeamMember,
    MemberRole,
    DEFAULT_MEMBERS,
    getStoredMembers,
    saveStoredMembers,
} from "@/lib/teamStore";

const roleColors: Record<string, string> = {
    "Team Lead": "text-warning bg-warning/10 border-warning/40 font-semibold",
    Admin: "text-primary bg-primary/10 border-primary/30",
    "Security Lead": "text-tertiary bg-tertiary/10 border-tertiary/30",
    Developer: "text-secondary bg-secondary/10 border-secondary/30",
    "DevOps Engineer": "text-primary bg-primary/10 border-primary/30",
    "CI Agent": "text-on-surface-variant bg-surface-container border-outline-variant/30",
};

export default function TeamPage() {
    const [members, setMembers] = useState<TeamMember[]>(DEFAULT_MEMBERS);
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [editingMemberId, setEditingMemberId] = useState<string | null>(null);

    // Form state for adding/editing
    const [formName, setFormName] = useState("");
    const [formEmail, setFormEmail] = useState("");
    const [formRole, setFormRole] = useState<MemberRole>("Developer");
    const [formGithub, setFormGithub] = useState("");
    const [formIsLead, setFormIsLead] = useState(false);

    useEffect(() => {
        setMembers(getStoredMembers());
    }, []);

    const updateMembersState = (newMembers: TeamMember[]) => {
        setMembers(newMembers);
        saveStoredMembers(newMembers);
    };

    const handleSetLead = (memberId: string) => {
        const updated = members.map((m) => ({
            ...m,
            isLead: m.id === memberId,
            role: m.id === memberId ? ("Team Lead" as MemberRole) : m.role === "Team Lead" ? ("Developer" as MemberRole) : m.role,
        }));
        updateMembersState(updated);
    };

    const handleRoleChange = (memberId: string, newRole: MemberRole) => {
        const updated = members.map((m) => {
            if (m.id !== memberId) {
                // If this member was set to Team Lead, clear others
                if (newRole === "Team Lead") {
                    return { ...m, isLead: false };
                }
                return m;
            }
            return {
                ...m,
                role: newRole,
                isLead: newRole === "Team Lead" ? true : m.isLead,
            };
        });
        updateMembersState(updated);
    };

    const handleDeleteMember = (memberId: string) => {
        if (members.length <= 1) {
            alert("Workspace must have at least one member.");
            return;
        }
        const updated = members.filter((m) => m.id !== memberId);
        // If the deleted member was lead, assign the first remaining as lead
        if (!updated.some((m) => m.isLead) && updated.length > 0) {
            updated[0].isLead = true;
            updated[0].role = "Team Lead";
        }
        updateMembersState(updated);
    };

    const openAddModal = () => {
        setFormName("");
        setFormEmail("");
        setFormRole("Developer");
        setFormGithub("@member");
        setFormIsLead(false);
        setIsAddModalOpen(true);
    };

    const handleSaveNewMember = (e: React.FormEvent) => {
        e.preventDefault();
        if (!formName.trim() || !formEmail.trim()) return;

        const initials = formName
            .trim()
            .split(" ")
            .map((p) => p[0])
            .join("")
            .slice(0, 2)
            .toUpperCase() || "MB";

        const newId = Date.now().toString();
        let updatedList = [...members];

        if (formIsLead || formRole === "Team Lead") {
            updatedList = updatedList.map((m) => ({
                ...m,
                isLead: false,
                role: m.role === "Team Lead" ? ("Developer" as MemberRole) : m.role,
            }));
        }

        const newMember: TeamMember = {
            id: newId,
            name: formName.trim(),
            email: formEmail.trim(),
            role: formIsLead ? "Team Lead" : formRole,
            isLead: formIsLead || formRole === "Team Lead",
            avatar: initials,
            github: formGithub.startsWith("@") ? formGithub : `@${formGithub}`,
            lastActive: "Just now",
            color: "from-primary-container to-secondary",
        };

        updatedList.unshift(newMember);
        updateMembersState(updatedList);
        setIsAddModalOpen(false);
    };

    const startEditing = (m: TeamMember) => {
        setEditingMemberId(m.id);
        setFormName(m.name);
        setFormEmail(m.email);
        setFormGithub(m.github || "");
        setFormRole(m.role);
    };

    const saveEditing = (memberId: string) => {
        const initials = formName
            .trim()
            .split(" ")
            .map((p) => p[0])
            .join("")
            .slice(0, 2)
            .toUpperCase() || "MB";

        const updated = members.map((m) => {
            if (m.id !== memberId) return m;
            return {
                ...m,
                name: formName.trim(),
                email: formEmail.trim(),
                github: formGithub.startsWith("@") ? formGithub : `@${formGithub}`,
                avatar: initials,
            };
        });
        updateMembersState(updated);
        setEditingMemberId(null);
    };

    const cancelEditing = () => {
        setEditingMemberId(null);
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                    <div className="flex items-center gap-3">
                        <h1 className="text-headline-xl font-bold text-on-surface">Team Management</h1>
                        <span className="px-2 py-0.5 rounded-lg bg-surface-container-high text-primary font-code-sm text-code-sm border border-outline-variant/40 font-medium">
                            {members.length} Members
                        </span>
                    </div>
                    <p className="mt-1 text-body-md text-on-surface-variant">
                        Select team lead and manage workspace members with customizable roles.
                    </p>
                </div>

                <div className="flex items-center gap-2">
                    <button
                        type="button"
                        onClick={openAddModal}
                        className="flex items-center gap-2 rounded-lg bg-primary-container px-3.5 py-2 text-body-sm font-semibold text-on-primary hover:bg-primary-container/90 transition shadow-lg shadow-primary-container/20 cursor-pointer"
                    >
                        <UserPlus className="h-4 w-4" />
                        <span>Add Member</span>
                    </button>
                </div>
            </div>

            {/* Team Lead Highlight Banner */}
            {(() => {
                const currentLead = members.find((m) => m.isLead) || members[0];
                return (
                    <div className="rounded-xl border border-warning/30 bg-warning/5 p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                        <div className="flex items-center gap-3.5">
                            <div className="h-12 w-12 rounded-full bg-gradient-to-br from-warning/30 to-warning-container/30 border border-warning/40 flex items-center justify-center font-bold text-base text-warning shrink-0">
                                <Crown className="h-6 w-6 text-warning" />
                            </div>
                            <div>
                                <div className="flex items-center gap-2">
                                    <span className="font-label-sm text-label-sm uppercase tracking-wider text-warning font-semibold">
                                        Current Team Lead
                                    </span>
                                    <span className="text-warning text-xs font-mono">({currentLead?.github})</span>
                                </div>
                                <h3 className="text-headline-md font-bold text-on-surface">
                                    {currentLead?.name}
                                </h3>
                                <p className="text-body-sm text-on-surface-variant">{currentLead?.email}</p>
                            </div>
                        </div>

                        <div className="flex items-center gap-2 text-label-sm text-on-surface-variant bg-surface-container-low px-3 py-2 rounded-lg border border-outline-variant/30">
                            <Star className="h-4 w-4 text-warning" />
                            <span>This member represents the active workspace lead in the sidebar and navigation.</span>
                        </div>
                    </div>
                );
            })()}

            {/* Members List */}
            <div className="space-y-3">
                <div className="flex items-center justify-between px-1 text-label-sm text-on-surface-variant font-medium">
                    <span>MEMBER DETAILS &amp; GITHUB</span>
                    <span>ROLE &amp; ACTIONS</span>
                </div>

                {members.map((member) => {
                    const isEditing = editingMemberId === member.id;

                    return (
                        <div
                            key={member.id}
                            className={`rounded-xl border transition-all p-4 ${
                                member.isLead
                                    ? "border-warning/40 bg-surface-container-low shadow-sm"
                                    : "border-outline-variant/30 bg-surface-container-low hover:bg-surface-container"
                            }`}
                        >
                            <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
                                {/* Left Section: Avatar + Name / Inputs */}
                                <div className="flex items-center gap-3.5 flex-1 min-w-0">
                                    <div
                                        className={`h-11 w-11 rounded-full bg-gradient-to-br ${
                                            member.color || "from-primary-container to-secondary"
                                        } flex items-center justify-center text-sm font-bold text-on-primary shrink-0 relative`}
                                    >
                                        {member.avatar}
                                        {member.isLead && (
                                            <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-warning flex items-center justify-center text-on-warning ring-2 ring-surface">
                                                <Crown className="h-3 w-3 text-black" />
                                            </span>
                                        )}
                                    </div>

                                    {isEditing ? (
                                        <div className="flex-1 grid grid-cols-1 sm:grid-cols-3 gap-2">
                                            <input
                                                type="text"
                                                value={formName}
                                                onChange={(e) => setFormName(e.target.value)}
                                                placeholder="Full Name"
                                                className="px-2.5 py-1.5 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                            />
                                            <input
                                                type="email"
                                                value={formEmail}
                                                onChange={(e) => setFormEmail(e.target.value)}
                                                placeholder="Email Address"
                                                className="px-2.5 py-1.5 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                            />
                                            <input
                                                type="text"
                                                value={formGithub}
                                                onChange={(e) => setFormGithub(e.target.value)}
                                                placeholder="@github"
                                                className="px-2.5 py-1.5 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                            />
                                        </div>
                                    ) : (
                                        <div className="min-w-0 flex-1">
                                            <div className="flex items-center gap-2">
                                                <h3 className="text-body-lg font-semibold text-on-surface truncate">
                                                    {member.name}
                                                </h3>
                                                {member.isLead && (
                                                    <span className="inline-flex items-center gap-1 rounded bg-warning/15 px-2 py-0.5 text-xs font-semibold text-warning border border-warning/30">
                                                        <Crown className="h-3 w-3" /> Team Lead
                                                    </span>
                                                )}
                                            </div>
                                            <div className="flex flex-wrap items-center gap-3 mt-1 text-label-sm text-on-surface-variant/70">
                                                <span className="flex items-center gap-1">
                                                    <Mail className="h-3 w-3" />
                                                    {member.email}
                                                </span>
                                                {member.github && (
                                                    <span className="flex items-center gap-1 font-mono text-primary">
                                                        <Github className="h-3 w-3" />
                                                        {member.github}
                                                    </span>
                                                )}
                                                <span className="flex items-center gap-1 text-on-surface-variant/50">
                                                    <Clock className="h-3 w-3" />
                                                    {member.lastActive}
                                                </span>
                                            </div>
                                        </div>
                                    )}
                                </div>

                                {/* Right Section: Role Select & Action Buttons */}
                                <div className="flex items-center gap-3 shrink-0">
                                    {isEditing ? (
                                        <div className="flex items-center gap-2">
                                            <button
                                                type="button"
                                                onClick={() => saveEditing(member.id)}
                                                className="p-1.5 rounded-lg bg-secondary-container hover:bg-secondary text-on-secondary-container hover:text-on-secondary transition"
                                                title="Save changes"
                                            >
                                                <Check className="h-4 w-4" />
                                            </button>
                                            <button
                                                type="button"
                                                onClick={cancelEditing}
                                                className="p-1.5 rounded-lg bg-surface-container hover:bg-surface-variant text-on-surface transition"
                                                title="Cancel"
                                            >
                                                <X className="h-4 w-4" />
                                            </button>
                                        </div>
                                    ) : (
                                        <>
                                            {/* Role Selector Dropdown */}
                                            <div className="flex items-center gap-2">
                                                <select
                                                    value={member.role}
                                                    onChange={(e) =>
                                                        handleRoleChange(member.id, e.target.value as MemberRole)
                                                    }
                                                    className={`rounded-lg border px-2.5 py-1 text-label-sm font-medium transition-colors cursor-pointer bg-surface-container-high focus:outline-none ${
                                                        roleColors[member.role] || "text-on-surface"
                                                    }`}
                                                >
                                                    <option value="Team Lead">Team Lead</option>
                                                    <option value="Security Lead">Security Lead</option>
                                                    <option value="Developer">Developer</option>
                                                    <option value="DevOps Engineer">DevOps Engineer</option>
                                                    <option value="Admin">Admin</option>
                                                    <option value="CI Agent">CI Agent</option>
                                                </select>

                                                {/* Select as Lead button if not already lead */}
                                                {!member.isLead && (
                                                    <button
                                                        type="button"
                                                        onClick={() => handleSetLead(member.id)}
                                                        className="px-2.5 py-1 rounded-lg border border-outline-variant/30 bg-surface-container hover:bg-warning/20 hover:border-warning/40 hover:text-warning text-on-surface-variant font-label-sm text-label-sm transition flex items-center gap-1 cursor-pointer"
                                                        title="Set as workspace Team Lead"
                                                    >
                                                        <Crown className="h-3.5 w-3.5 text-warning" />
                                                        <span>Set as Lead</span>
                                                    </button>
                                                )}
                                            </div>

                                            {/* Edit Member */}
                                            <button
                                                type="button"
                                                onClick={() => startEditing(member)}
                                                className="p-1.5 rounded-lg hover:bg-surface-container text-on-surface-variant hover:text-on-surface transition"
                                                title="Edit member details"
                                            >
                                                <Edit2 className="h-4 w-4" />
                                            </button>

                                            {/* Delete Member */}
                                            <button
                                                type="button"
                                                onClick={() => handleDeleteMember(member.id)}
                                                className="p-1.5 rounded-lg hover:bg-error/20 text-on-surface-variant hover:text-error transition"
                                                title="Remove member"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </button>
                                        </>
                                    )}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>

            {/* Add Member Modal */}
            {isAddModalOpen && (
                <div
                    className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4"
                    onClick={() => setIsAddModalOpen(false)}
                >
                    <div
                        className="bg-surface-container-low border border-outline-variant/60 rounded-xl max-w-md w-full p-6 shadow-2xl relative"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <div className="flex items-center justify-between pb-3 border-b border-outline-variant/30">
                            <h2 className="font-headline-md text-headline-md font-semibold text-on-surface flex items-center gap-2">
                                <UserPlus className="h-5 w-5 text-primary" />
                                Add Team Member
                            </h2>
                            <button
                                type="button"
                                onClick={() => setIsAddModalOpen(false)}
                                className="p-1 rounded hover:bg-surface-container text-on-surface-variant hover:text-on-surface"
                            >
                                <X className="h-5 w-5" />
                            </button>
                        </div>

                        <form onSubmit={handleSaveNewMember} className="mt-4 space-y-4">
                            <div>
                                <label className="block text-label-sm font-medium text-on-surface-variant mb-1">
                                    Full Name
                                </label>
                                <input
                                    type="text"
                                    required
                                    value={formName}
                                    onChange={(e) => setFormName(e.target.value)}
                                    placeholder="e.g. Alex Morgan"
                                    className="w-full px-3 py-2 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                />
                            </div>

                            <div>
                                <label className="block text-label-sm font-medium text-on-surface-variant mb-1">
                                    Email Address
                                </label>
                                <input
                                    type="email"
                                    required
                                    value={formEmail}
                                    onChange={(e) => setFormEmail(e.target.value)}
                                    placeholder="e.g. alex@lucid-security.org"
                                    className="w-full px-3 py-2 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                />
                            </div>

                            <div>
                                <label className="block text-label-sm font-medium text-on-surface-variant mb-1">
                                    GitHub Handle
                                </label>
                                <input
                                    type="text"
                                    value={formGithub}
                                    onChange={(e) => setFormGithub(e.target.value)}
                                    placeholder="e.g. @alex-sec"
                                    className="w-full px-3 py-2 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                />
                            </div>

                            <div>
                                <label className="block text-label-sm font-medium text-on-surface-variant mb-1">
                                    Member Role
                                </label>
                                <select
                                    value={formRole}
                                    onChange={(e) => setFormRole(e.target.value as MemberRole)}
                                    className="w-full px-3 py-2 rounded-lg bg-surface-container border border-outline-variant/40 text-on-surface text-body-sm focus:outline-none focus:border-primary"
                                >
                                    <option value="Team Lead">Team Lead</option>
                                    <option value="Security Lead">Security Lead</option>
                                    <option value="Developer">Developer</option>
                                    <option value="DevOps Engineer">DevOps Engineer</option>
                                    <option value="Admin">Admin</option>
                                    <option value="CI Agent">CI Agent</option>
                                </select>
                            </div>

                            <div className="flex items-center gap-2 pt-1">
                                <input
                                    type="checkbox"
                                    id="setAsLeadCheckbox"
                                    checked={formIsLead || formRole === "Team Lead"}
                                    onChange={(e) => setFormIsLead(e.target.checked)}
                                    className="rounded border-outline-variant bg-surface-container text-primary focus:ring-0"
                                />
                                <label htmlFor="setAsLeadCheckbox" className="text-body-sm text-on-surface cursor-pointer">
                                    Set as Workspace Team Lead (Crown)
                                </label>
                            </div>

                            <div className="flex items-center justify-end gap-3 pt-4 border-t border-outline-variant/30">
                                <button
                                    type="button"
                                    onClick={() => setIsAddModalOpen(false)}
                                    className="px-3.5 py-1.5 rounded-lg border border-outline-variant/40 bg-surface-container hover:bg-surface-container-high text-on-surface text-label-sm font-medium transition"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className="px-4 py-1.5 rounded-lg bg-primary-container text-on-primary hover:bg-primary-container/90 text-label-sm font-semibold transition shadow-md shadow-primary-container/20"
                                >
                                    Add Member
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}
