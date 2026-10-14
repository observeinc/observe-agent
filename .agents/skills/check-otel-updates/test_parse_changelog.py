"""Run: python3 -m unittest discover .agents/skills/check-otel-updates"""
import json
import os
import subprocess
import sys
import tempfile
import unittest

SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "parse_changelog.py")

GO_MOD = """module example
require (
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.151.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza v0.151.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.151.0
	go.opentelemetry.io/collector/service v0.151.0
)
"""

CONTRIB = """### Images and binaries here: https://example.invalid

## Unmaintained Components

- receiver/foo
- `receiver/host_metrics`: listed while unmaintained (#2)

## End User Changelog

### 🛑 Breaking changes 🛑

- `receiver/host_metrics`: Make the cpu attribute opt-in. (#10)
  First continuation line.
  Second continuation line.
- `pkg/fileconsumer`: Move a gate to beta. (#11)
- `pkg/stanza/operator/input/tcp`: Subpath of a module we link. (#12)
- `receiver/notours`: Component we don't build. (#13)
- `exporter/kafka` bullet without a colon after the tag

## API Changelog

### 🛑 Breaking changes 🛑

- `pkg/fileconsumer`: Move a gate to beta. (#11)
- `pkg/ottl`: API-only change. (#14)
"""

CORE = """## End User Changelog

### 💡 Enhancements 💡

- `pkg/service`: Core change. (#20)

## v1.0.0/v0.998.0

- `pkg/service`: Repeated entry from the previous release. (#19)
"""


def run(notes, gomod, out):
    return subprocess.run([sys.executable, SCRIPT, notes, gomod, out], capture_output=True, text=True)


class ParseChangelogTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        d = self.tmp.name
        self.notes = os.path.join(d, "notes")
        self.out = os.path.join(d, "out")
        os.makedirs(self.notes)
        self.gomod = os.path.join(d, "go.mod")
        with open(self.gomod, "w") as f:
            f.write(GO_MOD)

    def tearDown(self):
        self.tmp.cleanup()

    def write_notes(self):
        for name, body in (("contrib-0.999.0.md", CONTRIB), ("core-0.999.0.md", CORE)):
            with open(os.path.join(self.notes, name), "w") as f:
                f.write(body)

    def ledger(self):
        self.write_notes()
        res = run(self.notes, self.gomod, self.out)
        self.assertEqual(res.returncode, 0, res.stderr)
        with open(os.path.join(self.out, "ledger.json")) as f:
            return res, json.load(f)

    def test_scope_aliases_and_exclusions(self):
        _, ledger = self.ledger()
        prs = sorted(e["prs"][0] for e in ledger)
        # 10: kind-suffix normalisation, 11: pkg/fileconsumer alias, 12: pkg/stanza subpath,
        # 14: API-only entry, 20: core pkg/*. Excluded: 2 (Unmaintained), 13 (not built), 19 (repeat block).
        self.assertEqual(prs, ["10", "11", "12", "14", "20"])

    def test_api_duplicate_is_dropped_in_favour_of_end_user(self):
        _, ledger = self.ledger()
        dup = [e for e in ledger if e["prs"] == ["11"]]
        self.assertEqual(len(dup), 1)
        self.assertEqual(dup[0]["block"], "End User")

    def test_continuation_lines_go_into_body(self):
        _, ledger = self.ledger()
        body = next(e for e in ledger if e["prs"] == ["10"])["body"]
        self.assertIn("First continuation line.", body)
        self.assertIn("Second continuation line.", body)

    def test_ids_are_contiguous(self):
        _, ledger = self.ledger()
        self.assertEqual([int(e["id"].rsplit("-", 1)[1]) for e in ledger], list(range(len(ledger))))

    def test_unparsed_bullets_are_reported(self):
        res, _ = self.ledger()
        with open(os.path.join(self.out, "unparsed.txt")) as f:
            lines = f.read().splitlines()
        self.assertEqual(len(lines), 2)
        self.assertTrue(any("[Unmaintained] - receiver/foo" in l for l in lines))
        self.assertTrue(any("[End User] - `exporter/kafka` bullet without a colon" in l for l in lines))
        self.assertIn("2 unparsed top-level bullets (1 outside Unmaintained sections)", res.stderr)

    def test_missing_notes_fail(self):
        res = run(self.notes, self.gomod, self.out)
        self.assertNotEqual(res.returncode, 0)
        self.assertIn("no release notes", res.stderr)

    def test_notes_without_entries_fail(self):
        with open(os.path.join(self.notes, "core-0.999.0.md"), "w") as f:
            f.write("## End User Changelog\n\nNothing here.\n")
        res = run(self.notes, self.gomod, self.out)
        self.assertNotEqual(res.returncode, 0)
        self.assertIn("no changelog entries parsed", res.stderr)


if __name__ == "__main__":
    unittest.main()
