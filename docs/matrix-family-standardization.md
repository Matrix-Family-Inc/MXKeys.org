Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

<!--
Project: Matrix Family Standards (git.matrix.family)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Sun 22 Jun 2026 00:35:00 UTC
Status: Created
-->

# Matrix Family — Project Standardization Checklist

## Quick pass for all projects

Use this checklist in each repo. One pass: headers → branding → language → build.

### 1. File headers (every source file)

**Remove:**

- `EasyProTech`, `easypro.tech`, `Telegram`, `t.me/...`
- `Dev:`, `Maintainer:`, `Role:`
- any personal names

**Use only this block** (adapt `Project:` per repo):

```text
/*
 * Project: <Project Name> (<domain or repo>)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: <UTC, e.g. Sun 21 Jun 2026 23:35:00 UTC>
 * Status: Created
 */
```

**By file type:**

- `.rs`, `.ts`, `.tsx`, `.css`, `.js` → block above
- `.md` → same block at the top
- `.html` → same lines inside `<!-- ... -->`
- `Cargo.toml` / `#` configs → same lines with `#` instead of ` * `

Bulk-apply in MXKeys:

```bash
bash scripts/apply-matrix-family-headers.sh
```

---

### 2. Branding in content (not only headers)

Replace everywhere (including `index.html`, JSON-LD, locales, footers):

| Remove | Replace with |
|--------|----------------|
| EasyProTech LLC | Matrix Family Inc. |
| easypro.tech | matrix.family |
| t.me/EasyProTech | remove or `@support:matrix.family` / `support@matrix.family` |

Also check: `<meta author>`, copyright, Organization schema, `robots.txt`, SVG/OG assets.

---

### 3. English only

- Code, comments, docs, README — **English only**
- Remove `ru.ts` / `uk.ts` and other Cyrillic locale files
- Remove `hreflang=ru`, `og:locale:alternate`, `"inLanguage": ["en","ru"]`
- Public landing may keep non-Cyrillic UI locales (de, fr, ja, …); dev docs stay English-only
- After changes: run the Cyrillic scan from section 5; it must return nothing outside this checklist file (exclude `node_modules`, `dist`, `target`)

---

### 4. Git (Matrix Family servers)

Matrix Family uses **SSH port 42224 everywhere** — never port 22 for `git.matrix.family`.

```text
Web:  https://git.matrix.family/dev/MXKeys.org
SSH:  git@git.matrix.family:dev/MXKeys.org.git
Port: 42224 (not 22)
```

`~/.ssh/config`:

```text
Host git.matrix.family
    HostName git.matrix.family
    Port 42224
    User git
    IdentityFile ~/.ssh/<deploy_key>
    IdentitiesOnly yes
```

Clone:

```bash
git clone ssh://git@git.matrix.family:42224/dev/MXKeys.org.git
```

Check: `ssh -T -p 42224 git@git.matrix.family`

Production checkout on mxkeys.org: `/opt/MXKeys.org` (source tree).

---

### 5. One-liner searches (run in repo root)

```bash
# Old branding
rg -i 'EasyProTech|easypro\.tech|t\.me/EasyProTech|Telegram:' --glob '!node_modules' --glob '!dist' --glob '!target'

# Old header fields
rg 'Maintainer:|Role: Lead|Dev: Brabus' --glob '!node_modules' --glob '!dist'

# Cyrillic in source (use the Cyrillic Unicode letter-class in rg)
rg '<cyrillic-letter-class>' --glob '!node_modules' --glob '!dist' --glob '!target' --glob '!docs/matrix-family-standardization.md'
```

All three should be empty before you commit.

---

### 6. After edits

```bash
# frontend
cd landing && bun install && bun run build

# backend
go build -o mxkeys ./cmd/mxkeys

# production deploy (see docs/runbook/production-deploy.md)
bash scripts/deploy-landing.sh root@82.21.114.30 <version>
bash scripts/deploy-mxkeys.sh root@82.21.114.30 ./dist/mxkeys-<version>-linux-amd64 <version>

# optional on prod host
systemctl restart mxkeys
nginx -t && systemctl reload nginx
```

---

### 7. Commit message (example)

```text
chore: standardize Matrix Family headers and English-only docs
```

---

### MXKeys production layout

| Artefact | Path |
|----------|------|
| Source (git) | `/opt/MXKeys.org/` |
| Landing build (nginx root) | `/var/www/mxkeys.org/` |
| Go binary + config | `/opt/mxkeys/` |
| Signing keys | `/var/lib/mxkeys/keys/` |

---

**Order per repo:** headers → content branding → remove Cyrillic locales → search checks → build → commit.

See also: [ADR-0006](../adr/0006-file-header-standard.md), [production deploy runbook](runbook/production-deploy.md).
