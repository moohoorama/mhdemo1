"""uv run --with numpy --with pillow python -m unittest ver5/tools/test_snow_props.py"""
import sys
import unittest
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).parent))
import snow_props as sp


def px(*c):
    return np.array([[list(c) + [255]]], np.uint8)


class SnowProps(unittest.TestCase):
    def test_green_turns_white_bark_grey_outline_dark_alpha_kept(self):
        lit, shade = sp.snow(px(204, 214, 104))[0, 0, :3], sp.snow(px(34, 48, 26))[0, 0, :3]
        self.assertTrue((lit > 240).all())
        self.assertTrue(shade[2] > shade[0] and shade.max() < 150)
        bark = sp.snow(px(108, 88, 60))[0, 0, :3].astype(int)
        self.assertLess(bark.max() - bark.min(), 25)
        self.assertEqual(tuple(sp.snow(px(11, 16, 27))[0, 0, :3]), sp.PROFILES['tree']['outline'])
        clear = np.zeros((1, 1, 4), np.uint8)
        self.assertEqual(sp.snow(clear)[0, 0, 3], 0)
        self.assertEqual(sp.snow(px(240, 130, 160)).tolist(), px(240, 130, 160).tolist())

    def test_bamboo_leaf_keeps_a_green_cast_and_stem_is_darker_than_leaf(self):
        leaf = sp.snow(px(121, 184, 102), profile='bamboo')[0, 0, :3].astype(int)
        stem = sp.snow(px(104, 134, 48), profile='bamboo')[0, 0, :3].astype(int)
        self.assertGreater(leaf[1], leaf[0])
        self.assertGreater(leaf.sum(), stem.sum())

    def test_amount_and_determinism(self):
        a = np.random.default_rng(1).integers(0, 256, (8, 8, 4), dtype=np.uint8)
        self.assertTrue((sp.snow(a) == sp.snow(a)).all())
        self.assertTrue((sp.snow(a, 0) == a).all())


if __name__ == '__main__':
    unittest.main()
