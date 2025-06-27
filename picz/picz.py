from datetime import datetime
import json
from operator import attrgetter
import re
import sys

"""
To generate the metadata with exiftool, run

./exiftool -DateTimeOriginal -Model -FocalLengthIn35mmFormat -LensSpec -LensID \
    -LensInfo -Aperture -ISO -ShutterSpeed -FocalLength -ImageSize -Quality \
    -ShutterCount -ExposureCompensation -Make \
    -j -ext ARW -ext RW2 -ext NEF -ext rw2 -r \
    -- /Volumes/Photos/Pictures/ |tee ~/tmp/allpics_20240610.json
"""
def focal_length(l):
    if isinstance(l, (int, float)):
        return float(l)
    mo = re.match(r'(\d+(\.\d*)?)( mm)?', l)
    return float(mo.group(1))


class PhotoInfo:

    def __init__(self, json_data):
        self.created = datetime.strptime(json_data['DateTimeOriginal'], '%Y:%m:%d %H:%M:%S')
        self.path = json_data['SourceFile']
        # Camera
        self.make = json_data['Make']
        self.model = json_data['Model']
        # Lens
        self.focal_length = focal_length(json_data['FocalLength'])
        self.focal_length_35mm = focal_length(json_data['FocalLengthIn35mmFormat'])
        self.lens_info = None
        if 'LensID' in json_data:
            self.lens_info = json_data['LensID']
        elif 'LensInfo' in json_data:
            self.lens_info = json_data['LensInfo']
        self.lens_spec = json_data.get('LensSpec')
        # Exposure
        self.aperture = json_data['Aperture']
        self.iso = json_data['ISO']
        self.shutter_speed = json_data['ShutterSpeed']
        # Image size
        self.image_size = json_data['ImageSize']


def load_pics(paths):
    pics = []
    for path in paths:
        with open(path) as f_in:
            ps = json.load(f_in)
            pics.extend(ps)
    return sorted([
        PhotoInfo(p) for p in pics
    ], key=attrgetter('created'))


def main():
    pics = load_pics(sys.argv[1:])
    print(f"Loaded info for {len(pics)} photos.")
    print([p.created for p in pics])


if __name__ == '__main__':
    main()