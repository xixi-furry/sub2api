import unittest
from version import next_tag, validate_tag

class VersionTests(unittest.TestCase):
    def test_first_revision_and_new_upstream_reset(self):
        self.assertEqual(next_tag("0.2.7", ["v0.2.7", "v0.2.7-fur.1"]), "v0.2.7-fix1")
        self.assertEqual(next_tag("0.2.8", ["v0.2.7-fix9"]), "v0.2.8-fix1")

    def test_revisions_are_numeric_and_ignore_unrelated_tags(self):
        self.assertEqual(next_tag("0.2.7", ["v0.2.7-fix9", "v0.2.7-fix10", "v9.0.0-fix99"]), "v0.2.7-fix11")

    def test_only_matching_stable_base_is_accepted(self):
        self.assertEqual(validate_tag("v0.2.7-fix1", "0.2.7"), "0.2.7-fix1")
        for tag in ["v0.2.8-fix1", "v0.2.7", "v0.2.7-fix0", "v0.2.7-fix01", "v0.2.7-fix1;echo bad"]:
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                validate_tag(tag, "0.2.7")
        with self.assertRaises(ValueError):
            next_tag("0.2.7-rc1", [])

if __name__ == "__main__":
    unittest.main()
