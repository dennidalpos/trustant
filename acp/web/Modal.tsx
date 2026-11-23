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

import React, { useEffect, useId, useRef } from "react";

const FOCUSABLE =
	'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), ' +
	'select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Where Tab or Shift+Tab lands inside a modal with `count` focusable controls,
 * or null when the browser's own move stays inside. Pure so it can be tested
 * without a DOM.
 */
export function wrapFocusIndex(
	count: number,
	current: number,
	shift: boolean,
): number | null {
	if (count === 0) return -1;
	if (current === -1) return shift ? count - 1 : 0;
	if (shift && current === 0) return count - 1;
	if (!shift && current === count - 1) return 0;
	return null;
}

interface ModalProps {
	title: string;
	/** Called for a click on the backdrop and for Escape. */
	onClose: () => void;
	className?: string;
	children: React.ReactNode;
}

/**
 * Accessible modal dialog: role, label, focus moved in on open and restored on
 * close, Tab kept inside, Escape and backdrop click close it. A control that
 * already took focus inside the modal (for example an autoFocus input) keeps it.
 */
export function Modal({ title, onClose, className, children }: ModalProps) {
	const titleId = useId();
	const dialogRef = useRef<HTMLDivElement>(null);
	const onCloseRef = useRef(onClose);
	onCloseRef.current = onClose;

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;
		const returnTo = document.activeElement as HTMLElement | null;
		if (!dialog.contains(document.activeElement)) {
			const first = dialog.querySelector<HTMLElement>(FOCUSABLE);
			(first ?? dialog).focus();
		}
		return () => {
			if (returnTo && document.contains(returnTo)) returnTo.focus();
		};
	}, []);

	const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
		if (e.key === "Escape") {
			e.stopPropagation();
			onCloseRef.current();
			return;
		}
		if (e.key !== "Tab") return;
		const dialog = dialogRef.current;
		if (!dialog) return;
		const items = Array.from(dialog.querySelectorAll<HTMLElement>(FOCUSABLE));
		const target = wrapFocusIndex(
			items.length,
			items.indexOf(document.activeElement as HTMLElement),
			e.shiftKey,
		);
		if (target === null) return;
		e.preventDefault();
		(items[target] ?? dialog).focus();
	};

	return (
		<div className="modal-backdrop" onClick={onClose}>
			<div
				ref={dialogRef}
				className={className ? `modal ${className}` : "modal"}
				role="dialog"
				aria-modal="true"
				aria-labelledby={titleId}
				tabIndex={-1}
				onClick={(e) => e.stopPropagation()}
				onKeyDown={onKeyDown}
			>
				<div className="modal-title" id={titleId}>
					{title}
				</div>
				{children}
			</div>
		</div>
	);
}
