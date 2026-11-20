/*
 * Copyright 2025-2026 Nuvolaris Inc
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 * Derived from obsidian-agent-client (Apache-2.0), Copyright 2025-2026 RAIT-09; modified by Nuvolaris Inc. (see legal/VENDORED.md).
 */

// Keep ESLint discovery in acp/ while the legacy rules and packages live in
// the optional Obsidian tooling installation.
export { default } from "./obsidian-legacy/eslint.config.mjs";
