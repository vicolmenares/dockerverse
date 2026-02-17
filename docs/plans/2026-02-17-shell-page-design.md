# Shell Page Design

**Date:** 2026-02-17
**Goal:** Add a dedicated `/shell` page to DockerVerse with multi-tab terminal support, inspired by Dockhand's terminal section but with a modern multi-tab UX.

---

## Context

DockerVerse already has:
- `Terminal.svelte` — a fully-featured xterm.js terminal (WebGL, 7 themes, search, font control, drag, fullscreen, reconnect, download)
- Backend endpoints: `GET /ws/terminal/:hostId/:containerId` (docker exec) and `GET /ws/ssh/:hostId` (host SSH)
- Both endpoints fully operational

What's missing: a dedicated route/page for shell access. Currently Terminal.svelte only exists as a floating popup launched from ContainerCard.

Dockhand's approach (single container, one terminal at a time) was studied and used as inspiration for the toolbar design.

---

## Chosen Approach: Dedicated Page with Tab Bar

New route `/shell` that embeds xterm.js terminals in a full-page layout with a tab bar for multiple simultaneous sessions.

---

## Architecture

### Files to create/modify

| File | Change |
|------|--------|
| `frontend/src/routes/shell/+page.svelte` | NEW — Shell page with tab state, toolbar, tab bar |
| `frontend/src/lib/components/Terminal.svelte` | ADD `mode?: "popup" \| "embedded"` prop |
| `frontend/src/routes/+layout.svelte` | ADD Shell nav item with SquareTerminal icon |

**Backend: no changes needed.** WebSocket endpoints already exist.

---

## Layout

```
fixed top-16 left-[var(--sidebar-w)] right-0 bottom-0
┌─ toolbar ───────────────────────────────────────────────────────┐
│ Host: [Raspberry Main ▾]  Container: [nginx ▾]  [SSH Host] [Open Shell] │
├─ tab bar (scrollable) ──────────────────────────────────────────┤
│ ● nginx@raspi1 ×   ○ redis@raspi1 ×   ● raspi1 SSH ×   [+]     │
├─ terminal area (flex-1) ────────────────────────────────────────┤
│                                                                  │
│   xterm.js embedded — active tab terminal                        │
│                                                                  │
├─ status bar ────────────────────────────────────────────────────┤
│ ● Connected  |  Tokyo Night  |  Ctrl+F: Search  Ctrl+L: Clear   │
└─────────────────────────────────────────────────────────────────┘
```

The page uses `position: fixed` with `left: var(--sidebar-w, 16rem)` and `transition: left 300ms` — same pattern as the Logs page, so it expands when the sidebar collapses.

---

## Toolbar

- **Host dropdown**: all configured hosts (same data as Logs page host selector)
- **Container dropdown**: searchable list of running containers for selected host
- **[SSH Host] button**: opens a new tab with SSH connection to the host itself (`/ws/ssh/:hostId`)
- **[Open Shell] button** (primary): opens new tab with docker exec (`/ws/terminal/:hostId/:containerId`)

---

## Tab Bar

Each tab displays:
- Icon: `Container` icon for exec sessions, `Server` icon for SSH sessions
- Name: `nginx@raspi1` (container exec) or `raspi1 SSH` (host SSH)
- Status dot: green = connected, blue pulse = connecting, gray = disconnected
- `×` button to close that terminal (disconnects WebSocket, destroys xterm instance)

Tab bar scrolls horizontally if many tabs open.
`[+]` button at end repeats last action (open new tab with same container/host).

---

## Terminal.svelte — Embedded Mode

Add prop: `mode?: "popup" | "embedded"` (default: `"popup"` for backwards compatibility).

In `embedded` mode:
- Render as `flex flex-col h-full` instead of `fixed w-[800px] h-[500px]`
- No drag handle, no position state
- No `X` close button (tab handles closing)
- No fullscreen toggle (page is already full-page)
- All other features kept: themes, search (Ctrl+F), font size, WebGL renderer, copy, download, reconnect, status bar

---

## Sidebar Nav Entry

```ts
{
  id: "shell",
  icon: SquareTerminal,   // from lucide-svelte
  label: "Shell",
  href: "/shell",
}
```

Position: after "Logs" in the sidebar list.

Badge: show count of active (connected) terminal sessions when > 0.

---

## Empty State

When no tabs are open:
```
      >_

  Select a container or host above to open a shell

  [Open Shell]  [SSH Host]
```

---

## UX Details

- Each tab maintains independent state: its own xterm instance, WebSocket, connection status
- Switching tabs is instant — xterm instances are kept alive (not destroyed on tab switch), just hidden with `display: none` / `display: block`
- Closing a tab disconnects the WebSocket and disposes the xterm instance
- If a connection drops, the tab dot turns gray and auto-reconnect applies (inherited from Terminal.svelte)
- Keyboard shortcuts: Ctrl+T = new tab (focus toolbar), Ctrl+W = close active tab

---

## Design Tokens (ui-ux-pro-max)

- Tab bar background: `bg-background-secondary` with `border-b border-border`
- Active tab: `border-b-2 border-primary text-foreground`
- Inactive tab: `text-foreground-muted hover:text-foreground`
- Terminal area background: follows selected theme (default: Tokyo Night `#1a1b26`)
- Connected dot: `bg-running` (green)
- Connecting dot: `bg-primary animate-pulse` (blue pulse)
- Disconnected dot: `bg-paused` (amber/gray)

---

## What Dockhand Does Better (opportunities we improve on)

| Feature | Dockhand | DockerVerse Shell |
|---------|----------|-------------------|
| Multiple sessions | ✗ one at a time | ✓ multi-tab |
| Themes | ✗ hardcoded dark | ✓ 7 themes |
| Search in terminal | ✗ | ✓ Ctrl+F |
| Download output | ✗ | ✓ |
| WebGL renderer | ✗ | ✓ |
| Host SSH | ✓ | ✓ |
| Shell type selector | ✓ Bash/sh/Zsh/Ash | auto-detect bash→sh |
