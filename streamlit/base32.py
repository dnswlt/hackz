from base64 import b32decode, b32encode
from datetime import datetime, timedelta
import sys
import zoneinfo

SWISS_TIMEZONE = zoneinfo.ZoneInfo("Europe/Zurich")

BASE32_CHARS = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

SCRAMBLE_KEY = 179424691
UNSCRAMBLE_KEY = pow(SCRAMBLE_KEY, -1, 2**40)

def unscramble(n: int):
    mask = (1 << 40) - 1
    return (n * UNSCRAMBLE_KEY) & mask

def scramble(n: int) -> int:
    mask = (1 << 40) - 1
    return (n * SCRAMBLE_KEY) & mask


def id(n: int) -> tuple[str, int]:
    s1 = scramble(n) 
    bs1 = s1.to_bytes(length=5, byteorder="big")
    t1 = b32encode(bs1).decode("ascii")
    bs2 = b32decode(t1.encode('ascii'))
    s2 = int.from_bytes(bs2, byteorder="big")
    n2 = unscramble(s2)
    return t1, n2

def base32(n: int, min_len: int = 8) -> str:
    """Encode n using BASE32_CHARS. Use 0 padding to ensure min_len."""
    if n < 0:
        raise ValueError("Cannot base32 encode a negative number")

    if n == 0:
        return BASE32_CHARS[0] * min_len

    base = len(BASE32_CHARS)
    result_chars = []

    while n > 0:
        remainder = n % base
        result_chars.append(BASE32_CHARS[remainder])
        n //= base

    # The characters are in reverse order (least significant first)
    s = "".join(reversed(result_chars))

    # Pad with '0' (the first char) to the left to meet min_len
    return s.rjust(min_len, BASE32_CHARS[0])


def capacity_id(otn: int, departure_time: datetime, network: str) -> str:
    """Reference implementation for computing the CapacityID in vBv."""
    if otn < 0:
        raise ValueError(f"Negative OTN: {otn}")
    otn = otn % 100000

    if departure_time.tzinfo != SWISS_TIMEZONE:
        raise ValueError(f"departure_time {departure_time} is not in Swiss timezone")
    if departure_time.year < 1900 or departure_time.year >= 3000:
        raise ValueError(f"Only years between 1900 and 2999 are supported")

    year_start = datetime(departure_time.year, 1, 1, 0, 0, 0, tzinfo=SWISS_TIMEZONE)
    d = departure_time - year_start
    hours = int(d.total_seconds()) // (60 * 60)
    repr_time = year_start + timedelta(hours=hours)

    if network.lower() == "standard":
        network_num = 0
    else:
        network_num = 1

    # Bits 0-3 (4 bits): network_num
    # Bits 4-20 (17 bits): OTN
    # Bits 21-34 (14 bits): hours
    print(f"Hours since T_0 (= {year_start}): {hours} ({repr_time})")
    print(f"OTN: {otn}")
    print(f"Network number: {network_num} (for '{network}')")

    n = hours * 2 ** (4 + 17) + otn * 2 ** (4) + network_num
    s = scramble(n) 

    return base32(s)


def main():
    departure_time = datetime.strptime(sys.argv[1], "%Y-%m-%dT%H:%M")
    departure_time = departure_time.replace(tzinfo=SWISS_TIMEZONE)
    otn = int(sys.argv[2])
    network = sys.argv[3]
    c_id = capacity_id(otn, departure_time, network)

    print(c_id)


if __name__ == "__main__":
    main()
