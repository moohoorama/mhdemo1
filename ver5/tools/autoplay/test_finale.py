"""Check that retries and truncated campaigns cannot count as a finale clear."""

from pathlib import Path
import tempfile
import unittest

from finale import completed, counts, progress


class FinaleTests(unittest.TestCase):
    def test_completion_requires_twenty_real_first_attempt_victories(self):
        report = {"Cleared": True, "ExpectedBattles": 20,
                  "Stages": [{"Won": True, "Tries": 1} for _ in range(20)]}
        self.assertTrue(completed(report))
        self.assertEqual(progress(report), 20)
        report["Cleared"] = False
        self.assertFalse(completed(report))
        report["Cleared"] = True
        report["Stages"][10]["Tries"] = 2
        self.assertEqual(progress(report), 10)
        self.assertFalse(completed(report))
        report["Stages"] = report["Stages"][:9]
        self.assertFalse(completed(report))

    def test_counts_separate_new_games_and_frontier_repeats(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            (base / "train.log").write_text(
                "collect 1 neural seed 1\n"
                "teacher 1/2 victory=true\n"
                "teacher decisions B13 40\n"
                "full campaign dev after 0 mean 7\n"
                "frontier 8/128 victory 3/8\n"
                "frontier 16/128 victory 4/8\n")
            (base / "verification").mkdir()
            (base / "verification" / "result.json").write_text("[]")
            self.assertEqual(counts(base), {
                "fresh_campaign_evaluations": 6,
                "frontier_training_battles": 16, "teacher_battles": 1})


if __name__ == "__main__":
    unittest.main()
