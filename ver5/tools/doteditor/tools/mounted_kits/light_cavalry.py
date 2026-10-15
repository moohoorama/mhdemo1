"""경기병: the rider with a faction hood instead of an iron helmet, steel shoulders and shins, spear."""
import grid3d_mounted as M


def make():
    return M.rider_build('rider'), M.Hd.heads(M.Hd.HOOD, M.Hd.HOOD_CLOTH)


M.register('light_cavalry', make, vertical_spear=True)
