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

// Modal accessibility for every `.nu-modal-scrim` dialog.
//
// Pages show and hide modals by toggling the `hidden` class on the scrim. This
// script observes that class and, without touching page logic:
//   - gives each modal role="dialog", aria-modal and a label from its heading;
//   - moves focus into the modal when it opens and restores it when it closes;
//   - keeps Tab/Shift+Tab inside the topmost open modal.
(function () {
    'use strict';

    var FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), ' +
        'select:not([disabled]), textarea:not([disabled]), iframe:not([tabindex="-1"]), ' +
        '[tabindex]:not([tabindex="-1"]):not([data-modal-focus-end])';
    var opened = []; // stack of { modal, returnTo }
    var uid = 0;
    var wrappingBackward = false;

    function isOpen(el) { return !el.classList.contains('hidden'); }

    function visibleFocusables(root) {
        return Array.prototype.filter.call(root.querySelectorAll(FOCUSABLE), function (n) {
            return n.offsetParent !== null || n === document.activeElement;
        });
    }

    function label(modal) {
        if (modal.hasAttribute('aria-label') || modal.hasAttribute('aria-labelledby')) return;
        var heading = modal.querySelector('h1, h2, h3, h4');
        if (!heading) return;
        if (!heading.id) heading.id = (modal.id || 'modal') + 'Title' + (++uid);
        modal.setAttribute('aria-labelledby', heading.id);
    }

    function enhance(modal) {
        if (modal.dataset.a11y === '1') return;
        modal.dataset.a11y = '1';
        if (!modal.hasAttribute('role')) modal.setAttribute('role', 'dialog');
        modal.setAttribute('aria-modal', 'true');
        label(modal);
    }

    function onOpen(modal) {
        if (opened.some(function (o) { return o.modal === modal; })) return;
        label(modal);
        opened.push({ modal: modal, returnTo: document.activeElement });
        // Respect a focus the page already placed inside the modal.
        if (modal.contains(document.activeElement)) return;
        var target = modal.querySelector('[data-modal-initial-focus], [autofocus]') || visibleFocusables(modal)[0];
        if (!target) {
            var panel = modal.firstElementChild || modal;
            if (!panel.hasAttribute('tabindex')) panel.setAttribute('tabindex', '-1');
            target = panel;
        }
        target.focus();
    }

    function onClose(modal) {
        for (var i = opened.length - 1; i >= 0; i--) {
            if (opened[i].modal !== modal) continue;
            var back = opened[i].returnTo;
            opened.splice(i, 1);
            if (back && document.contains(back) && typeof back.focus === 'function') back.focus();
            return;
        }
    }

    function sync(modal) {
        if (isOpen(modal)) onOpen(modal); else onClose(modal);
    }

    var observer = new MutationObserver(function (records) {
        records.forEach(function (r) {
            if (r.type === 'attributes' && r.target.classList.contains('nu-modal-scrim')) sync(r.target);
        });
    });

    function watch(modal) {
        enhance(modal);
        // Tab events inside a frame do not bubble to the containing page.
        var end = modal.querySelector('[data-modal-focus-end]');
        if (end) end.addEventListener('focus', function () {
            if (wrappingBackward) return;
            var first = visibleFocusables(modal)[0];
            if (first) first.focus();
        });
        observer.observe(modal, { attributes: true, attributeFilter: ['class'] });
        if (isOpen(modal)) onOpen(modal);
    }

    document.addEventListener('keydown', function (e) {
        if (e.key !== 'Tab' || !opened.length) return;
        var top = opened[opened.length - 1].modal;
        var items = visibleFocusables(top);
        if (!items.length) { e.preventDefault(); return; }
        var first = items[0], last = items[items.length - 1];
        if (!top.contains(document.activeElement)) { e.preventDefault(); first.focus(); }
        else if (e.shiftKey && document.activeElement === first) {
            var end = top.querySelector('[data-modal-focus-end]');
            if (end) {
                // Let native Shift+Tab enter the last control of the frame.
                wrappingBackward = true;
                end.focus();
                wrappingBackward = false;
            } else {
                e.preventDefault();
                last.focus();
            }
        }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    });

    function init() {
        Array.prototype.forEach.call(document.querySelectorAll('.nu-modal-scrim'), watch);
    }
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
    else init();
})();
