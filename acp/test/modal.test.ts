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
 */

import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { Modal, wrapFocusIndex } from "../web/Modal";

describe("modal", () => {
	it("renders a labelled modal dialog", () => {
		const markup = renderToStaticMarkup(
			React.createElement(
				Modal,
				{
					title: "Pi sessions",
					onClose: () => {},
					className: "history-modal",
				},
				React.createElement("button", { type: "button" }, "Close"),
			),
		);

		expect(markup).toContain('role="dialog"');
		expect(markup).toContain('aria-modal="true"');
		const labelledBy = /aria-labelledby="([^"]+)"/.exec(markup)?.[1];
		expect(labelledBy).toBeTruthy();
		expect(markup).toContain(`id="${labelledBy}"`);
		expect(markup).toContain(">Pi sessions</div>");
		expect(markup).toContain('class="modal history-modal"');
		expect(markup).toContain('class="modal-backdrop"');
	});

	it("wraps Tab and Shift+Tab inside the dialog", () => {
		expect(wrapFocusIndex(3, 2, false)).toBe(0);
		expect(wrapFocusIndex(3, 0, true)).toBe(2);
		expect(wrapFocusIndex(3, 1, false)).toBeNull();
		expect(wrapFocusIndex(3, 1, true)).toBeNull();
		// Focus outside the dialog comes back to its ends.
		expect(wrapFocusIndex(3, -1, false)).toBe(0);
		expect(wrapFocusIndex(3, -1, true)).toBe(2);
		// No focusable control: keep focus on the dialog itself.
		expect(wrapFocusIndex(0, -1, false)).toBe(-1);
	});
});
