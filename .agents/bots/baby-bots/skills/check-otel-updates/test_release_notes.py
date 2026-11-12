import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))

import release_notes  # noqa: E402

CHANGELOG = """<!-- header -->
# Changelog

## v1.4.1/v0.165.1

### 🧰 Bug fixes 🧰

- `receiver/foo`: fix a thing (#3)

## v1.4.0/v0.165.0

### 🛑 Breaking changes 🛑

- `processor/bar`: rename a key (#2)
  more body

## v0.164.0

- `exporter/baz`: old (#1)
"""


class SlicesTest(unittest.TestCase):
    def test_maps_each_heading_to_its_body(self):
        s = release_notes.slices(CHANGELOG)
        self.assertEqual(sorted(s), ["0.164.0", "0.165.0", "0.165.1"])
        self.assertIn("rename a key (#2)\n  more body", s["0.165.0"])
        self.assertNotIn("#3", s["0.165.0"])

    def test_between_is_exclusive_low_inclusive_high_and_sorted(self):
        s = release_notes.slices(CHANGELOG)
        self.assertEqual(release_notes.between(s, "0.164.0", "0.165.1"), ["0.165.0", "0.165.1"])
        self.assertEqual(release_notes.between(s, "0.165.1", "0.166.0"), [])

    def test_notes_omit_an_absent_api_slice(self):
        self.assertNotIn("API", release_notes.notes("- `a`: b", ""))
        self.assertIn("## API Changelog", release_notes.notes("- `a`: b", "- `c`: d"))


class ParserRoundTripTest(unittest.TestCase):
    def test_parse_changelog_reads_the_rebuilt_notes(self):
        base = os.path.join(os.path.dirname(__file__), "base")
        sys.path.insert(0, base)
        import parse_changelog
        import tempfile

        s = release_notes.slices(CHANGELOG)
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "contrib-0.165.0.md")
            with open(path, "w") as f:
                f.write(release_notes.notes(s["0.165.0"], "- `processor/bar`: rename a key (#2)"))
            entries, unparsed, per_file = parse_changelog.parse([path])
        self.assertEqual(per_file["contrib-0.165.0.md"], 2)
        self.assertEqual({e["block"] for e in entries}, {"End User", "API"})
        self.assertEqual(entries[0]["heading"], "Breaking changes")
        self.assertEqual(unparsed, [])


if __name__ == "__main__":
    unittest.main()
