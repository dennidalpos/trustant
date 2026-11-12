# Copyright 2025-2026 Nuvolaris Inc
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

"""Offline checks for preserving legal evidence during regeneration."""

import contextlib
import importlib.util
import io
from pathlib import Path
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location(
    "trustant_deps", Path(__file__).resolve().parents[1] / "legal" / "deps.py")
deps = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(deps)


class LegalGenerationTests(unittest.TestCase):
    def component(self):
        return dict(section="web", name="fixture", version="1.0", license="MIT",
                    homepage="https://example.invalid/fixture", fetch="npm:fixture@1.0")

    def test_unavailable_source_preserves_all_existing_outputs(self):
        with patch.object(deps, "components", return_value=[self.component()]), \
                patch.object(deps, "fetch", return_value=None), \
                patch.object(deps, "write") as write, \
                contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                deps.main()
            write.assert_not_called()

    def test_missing_spdx_text_preserves_all_existing_outputs(self):
        with patch.object(deps, "components", return_value=[self.component()]), \
                patch.object(deps, "fetch", return_value="."), \
                patch.object(deps, "legal_files", return_value=(["LICENSE"], [])), \
                patch.object(deps, "read_text", return_value="Original legal text\n"), \
                patch.object(deps, "license_text", side_effect=SystemExit(1)), \
                patch.object(deps, "write") as write:
            with self.assertRaises(SystemExit):
                deps.main()
            write.assert_not_called()

    def test_shared_source_fetched_once_and_multiline_notice_preserved(self):
        original = "Copyright (c) 2026\n    Example Authors\nCustom attribution condition.\n"
        with patch.object(deps, "components", return_value=[self.component(), self.component()]), \
                patch.object(deps, "fetch", return_value=".") as fetch, \
                patch.object(deps, "legal_files", return_value=(["LICENSE"], ["NOTICE"])), \
                patch.object(deps, "read_text", return_value=original), \
                patch.object(deps, "license_text", return_value="SPDX fixture text"), \
                patch.object(deps, "write") as write, \
                contextlib.redirect_stdout(io.StringIO()):
            deps.main()
            fetch.assert_called_once()
            notice = next(c.args[1] for c in write.call_args_list if c.args[0] == "NOTICE")
            self.assertIn("  LICENSE FILE: LICENSE\n  | Copyright (c) 2026\n"
                          "  |     Example Authors\n  | Custom attribution condition.", notice)
            self.assertIn("  NOTICE:\n  | Copyright (c) 2026\n"
                          "  |     Example Authors\n  | Custom attribution condition.", notice)

    def test_ofl_is_a_license_file(self):
        self.assertIsNotNone(deps.LEGAL_FILE.fullmatch("OFL.txt"))
        self.assertIsNone(deps.LEGAL_FILE.fullmatch("README.md"))


if __name__ == "__main__":
    unittest.main()
